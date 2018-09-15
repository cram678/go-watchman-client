package watchman

import (
	"github.com/sjansen/watchman/protocol"
)

// A ChangeNotification represents changes two one or more filesystem entries.
type ChangeNotification struct {
	IsFreshInstance bool
	Clock           string
	Subscription    string
	Files           []File
}

func newChangeNotification(sub *protocol.Subscription) *ChangeNotification {
	clock := sub.Clock()
	files := sub.Files()
	cn := &ChangeNotification{
		IsFreshInstance: sub.IsFreshInstance(),
		Clock:           clock,
		Subscription:    sub.Subscription(),
		Files:           make([]File, len(files)),
	}
	for i, file := range files {
		f := &cn.Files[i]
		f.Name = file["name"].(string)
		f.Type = file["type"].(string)
		if f.Type == "l" {
			f.Target = file["symlink_target"].(string)
		}
		f.Size = int64(file["size"].(float64))
		cclock := file["cclock"].(string)
		exists := file["exists"].(bool)
		switch {
		case clock == cclock && exists:
			f.Change = Created
		case !exists:
			f.Change = Removed
		default:
			f.Change = Updated
		}
	}
	return cn
}

// A File represents changes in the state of a single filesystem entry.
type File struct {
	Change StateChange
	Name   string
	Type   string
	Target string
	Size   int64
}

// A StateChange describes how a file's state has changed.
type StateChange int

const (
	Created StateChange = 1 << iota
	Removed
	Updated
)

func (c StateChange) String() string {
	switch c {
	case Created:
		return "created"
	case Removed:
		return "removed"
	case Updated:
		return "updated"
	}
	return "unknown"
}
