package netcode

import "hockeyv2/internal/sim"

type MessageKind string

const (
	MessageJoinRequest     MessageKind = "join_request"
	MessageJoinAccepted    MessageKind = "join_accepted"
	MessageRoomListRequest MessageKind = "room_list_request"
	MessageRoomList        MessageKind = "room_list"
	MessageInputFrame      MessageKind = "input_frame"
	MessageSnapshot        MessageKind = "snapshot"
	MessageError           MessageKind = "error"
	MessagePing            MessageKind = "ping"
	MessagePong            MessageKind = "pong"
)

type RoomSummary struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Players  int    `json:"players"`
	Capacity int    `json:"capacity"`
}

func (r RoomSummary) Joinable() bool {
	if r.Capacity <= 0 {
		return true
	}
	return r.Players < r.Capacity
}

type Message struct {
	Kind       MessageKind    `json:"kind"`
	MatchID    string         `json:"match_id,omitempty"`
	ClientID   string         `json:"client_id,omitempty"`
	Team       sim.Team       `json:"team,omitempty"`
	Tick       uint64         `json:"tick,omitempty"`
	TickRate   int            `json:"tick_rate,omitempty"`
	Move       sim.Vec2       `json:"move,omitempty"`
	Shoot      bool           `json:"shoot,omitempty"`
	Pass       bool           `json:"pass,omitempty"`
	Switch     bool           `json:"switch,omitempty"`
	Ready      bool           `json:"ready,omitempty"`
	ColorPrev  bool           `json:"color_prev,omitempty"`
	ColorNext  bool           `json:"color_next,omitempty"`
	State      *sim.GameState `json:"state,omitempty"`
	Error      string         `json:"error,omitempty"`
	RoomCode   string         `json:"room_code,omitempty"`
	RoomName   string         `json:"room_name,omitempty"`
	Rooms      []RoomSummary  `json:"rooms,omitempty"`
	CreateRoom bool           `json:"create_room,omitempty"`
	Host       bool           `json:"host,omitempty"`
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
