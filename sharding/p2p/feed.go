package p2p

import (
	"reflect"

	"github.com/ethereum/go-ethereum/event"
)

// P2P feed is a one to many subscription feed of the argument type.
//
// Messages received via p2p protocol are sent to subscribers by these event
// feeds. Message consumers should not use event feeds to reply to or broadcast
// messages. The p2p server will not relay them to peers. Rather, use the
// Send() or Broadcast() method on p2p.Server.
//
// Event feeds from p2p will always be of type p2p.Message. The message
// contains information about the sender, aka the peer, and the message payload
// itself.
//
//   feed := ps.Feed(MyMessage{})
//   ch := make(chan p2p.Message, 100) // Choose a reasonable buffer size!
//   feed.Subscribe(ch)
//
//   // Wait until my message comes from a peer.
//   msg := <- ch
//   fmt.Printf("Message received: %v", msg.Data)
func (s *Server) Feed(msg interface{}) (*event.Feed, error) {
	t := reflect.TypeOf(msg)
	if s.feeds[t] == nil {
		s.feeds[t] = new(event.Feed)
	}
	return s.feeds[t], nil
}
