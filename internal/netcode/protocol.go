package netcode

import "hockeyv2/internal/sim"

type MessageKind string

const (
	MessageJoinRequest  MessageKind = "join_request"
	MessageJoinAccepted MessageKind = "join_accepted"
	MessageInputFrame   MessageKind = "input_frame"
	MessageSnapshot     MessageKind = "snapshot"
	MessageError        MessageKind = "error"
	MessagePing         MessageKind = "ping"
	MessagePong         MessageKind = "pong"
)

type Message struct {
	Kind      MessageKind    `json:"kind"`
	MatchID   string         `json:"match_id,omitempty"`
	ClientID  string         `json:"client_id,omitempty"`
	Team      sim.Team       `json:"team,omitempty"`
	Tick      uint64         `json:"tick,omitempty"`
	TickRate  int            `json:"tick_rate,omitempty"`
	Move      sim.Vec2       `json:"move,omitempty"`
	Shoot     bool           `json:"shoot,omitempty"`
	Pass      bool           `json:"pass,omitempty"`
	Switch    bool           `json:"switch,omitempty"`
	Ready     bool           `json:"ready,omitempty"`
	ColorPrev bool           `json:"color_prev,omitempty"`
	ColorNext bool           `json:"color_next,omitempty"`
	State     *sim.GameState `json:"state,omitempty"`
	Error     string         `json:"error,omitempty"`
}

func MessageFromInput(input sim.InputFrame, clientID string) Message {
	return Message{
		Kind:      MessageInputFrame,
		ClientID:  clientID,
		Team:      input.Team,
		Tick:      input.Tick,
		Move:      input.Move,
		Shoot:     input.Shoot,
		Pass:      input.Pass,
		Switch:    input.Switch,
		Ready:     input.Ready,
		ColorPrev: input.ColorPrev,
		ColorNext: input.ColorNext,
	}
}

func (m Message) ToInputFrame() sim.InputFrame {
	return sim.InputFrame{
		ClientID:  m.ClientID,
		Team:      m.Team,
		Tick:      m.Tick,
		Move:      m.Move,
		Shoot:     m.Shoot,
		Pass:      m.Pass,
		Switch:    m.Switch,
		Ready:     m.Ready,
		ColorPrev: m.ColorPrev,
		ColorNext: m.ColorNext,
	}
}
