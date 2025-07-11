// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"encoding/json"
	"strings"

	apiutil "github.com/absmach/supermq/api/http/util"
)

// State represents invitation state.
type State uint8

const (
	AllState State = iota // All is used for querying purposes to list invitations irrespective of their state - both pending and accepted.
	Pending               // Pending is the state of an invitation that has not been accepted yet.
	Accepted              // Accepted is the state of an invitation that has been accepted.
	Rejected              // Rejected is the state of an invitation that has been rejected.
)

// String representation of the possible state values.
const (
	all          = "all"
	pending      = "pending"
	accepted     = "accepted"
	rejected     = "rejected"
	UnknownState = "unknown"
)

// String converts invitation state to string literal.
func (s State) String() string {
	switch s {
	case AllState:
		return all
	case Pending:
		return pending
	case Accepted:
		return accepted
	case Rejected:
		return rejected
	default:
		return UnknownState
	}
}

// ToState converts string value to a valid invitation state.
func ToState(status string) (State, error) {
	switch status {
	case all:
		return AllState, nil
	case pending:
		return Pending, nil
	case accepted:
		return Accepted, nil
	case rejected:
		return Rejected, nil
	}

	return State(0), apiutil.ErrInvitationState
}

func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Custom Unmarshaler for Client/Groups.
func (s *State) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToState(str)
	*s = val
	return err
}
