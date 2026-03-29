package netcode

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"

	"hockeyv2/internal/sim"
)

type Client struct {
	conn      net.Conn
	encoder   *json.Encoder
	team      sim.Team
	clientID  string
	matchID   string
	snapshots chan sim.GameState
	errs      chan error
	sendMu    sync.Mutex
	closeOnce sync.Once
	closed    chan struct{}
}

func Dial(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bufio.NewReader(conn))
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(Message{Kind: MessageJoinRequest}); err != nil {
		conn.Close()
		return nil, err
	}

	var first Message
	if err := decoder.Decode(&first); err != nil {
		conn.Close()
		return nil, err
	}
	if first.Kind == MessageError {
		conn.Close()
		return nil, errors.New(first.Error)
	}
	if first.Kind != MessageJoinAccepted {
		conn.Close()
		return nil, fmt.Errorf("unexpected first message: %s", first.Kind)
	}

	client := &Client{
		conn:      conn,
		encoder:   encoder,
		team:      first.Team,
		clientID:  first.ClientID,
		matchID:   first.MatchID,
		snapshots: make(chan sim.GameState, 4),
		errs:      make(chan error, 1),
		closed:    make(chan struct{}),
	}
	if first.State != nil {
		deliverSnapshot(client.snapshots, cloneGameState(*first.State))
	}
	go client.readLoop(decoder)
	return client, nil
}

func (c *Client) Team() sim.Team {
	return c.team
}

func (c *Client) MatchID() string {
	return c.matchID
}

func (c *Client) ClientID() string {
	return c.clientID
}

func (c *Client) Snapshots() <-chan sim.GameState {
	return c.snapshots
}

func (c *Client) Errors() <-chan error {
	return c.errs
}

func (c *Client) SendInput(input sim.InputFrame) error {
	input.Team = c.team
	input.ClientID = c.clientID
	message := MessageFromInput(input, c.clientID)
	message.MatchID = c.matchID
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return c.encoder.Encode(message)
}

func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.conn.Close()
	})
	return err
}

func (c *Client) readLoop(decoder *json.Decoder) {
	for {
		var message Message
		if err := decoder.Decode(&message); err != nil {
			select {
			case <-c.closed:
				return
			default:
			}
			select {
			case c.errs <- err:
			default:
			}
			return
		}
		switch message.Kind {
		case MessageSnapshot:
			if message.State != nil {
				deliverSnapshot(c.snapshots, cloneGameState(*message.State))
			}
		case MessageError:
			select {
			case c.errs <- errors.New(message.Error):
			default:
			}
			return
		}
	}
}

func deliverSnapshot(ch chan sim.GameState, state sim.GameState) {
	select {
	case ch <- state:
		return
	default:
	}
	select {
	case <-ch:
	default:
	}
	select {
	case ch <- state:
	default:
	}
}

func cloneGameState(state sim.GameState) sim.GameState {
	copyState := state
	copyState.HomeSkaters = append([]sim.SkaterState(nil), state.HomeSkaters...)
	copyState.AwaySkaters = append([]sim.SkaterState(nil), state.AwaySkaters...)
	return copyState
}
