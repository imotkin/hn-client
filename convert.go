package hn

import "fmt"

var converters = map[string]func(Item) Convertible{
	StoryType: func(i Item) Convertible {
		return ToStory(i)
	},
	CommentType: func(i Item) Convertible {
		return ToComment(i)
	},
	AskType: func(i Item) Convertible {
		return ToAsk(i)
	},
	JobType: func(i Item) Convertible {
		return ToJob(i)
	},
	PollType: func(i Item) Convertible {
		return ToPoll(i)
	},
	PollOptionType: func(i Item) Convertible {
		return ToPollOption(i)
	},
}

// Convertible describes types that can be converted from an Item.
type Convertible interface {
	Type() string
}

// To is a helper function to convert any Item struct
// to a struct of a specific type that implements the Convertible interface:
// Comment, Story, Ask, Job, Poll or PollOption.
// Another alternative for this function is a specific converter (ToComment, ToStory, etc.).
func To[C Convertible](item Item) (c C, err error) {
	if item.Type != c.Type() {
		return c, fmt.Errorf("mismatched types. expected '%v', but got '%v'", c.Type(), item.Type)
	}

	fn, ok := converters[item.Type]
	if !ok {
		return c, fmt.Errorf("unsupported type: %v", item.Type)
	}

	c = fn(item).(C)

	return c, nil
}

// ToList converts a slice of items to a list of structs of a specific type.
//
// If the type of any item doesn't match the output type, the item is excluded from the converted list.
func ToList[C Convertible](items []Item) []C {
	list := make([]C, 0, len(items))

	for _, item := range items {
		val, err := To[C](item)
		if err != nil {
			continue
		}

		list = append(list, val)
	}

	return list
}

// ToComment converts an Item struct to a Comment struct.
func ToComment(item Item) Comment {
	return Comment{
		baseItem: item.baseItem,
		Kids:     item.Kids,
		Parent:   item.Parent,
		Text:     item.Text,
	}
}

// ToStory converts an Item struct to a Story struct.
func ToStory(item Item) Story {
	return Story{
		baseItem:    item.baseItem,
		Descendants: item.Descendants,
		Kids:        item.Kids,
		Text:        item.Text,
		Title:       item.Title,
		URL:         item.URL,
	}
}

// ToAsk converts an Item struct to an Ask struct.
func ToAsk(item Item) Ask {
	return Ask{
		baseItem:    item.baseItem,
		Descendants: item.Descendants,
		Kids:        item.Kids,
		Text:        item.Text,
		Title:       item.Title,
	}
}

// ToJob converts an Item struct to a Job struct.
func ToJob(item Item) Job {
	return Job{
		baseItem: item.baseItem,
		Text:     item.Text,
		Title:    item.Title,
		URL:      item.URL,
	}
}

// ToPoll converts an Item struct to a Poll struct.
func ToPoll(item Item) Poll {
	return Poll{
		baseItem:    item.baseItem,
		Descendants: item.Descendants,
		Kids:        item.Kids,
		Parts:       item.Parts,
		Text:        item.Text,
		Title:       item.Title,
	}
}

// ToPollOption converts an Item struct to a PollOption struct.
func ToPollOption(item Item) PollOption {
	return PollOption{
		baseItem: item.baseItem,
		Poll:     item.Poll,
		Text:     item.Text,
	}
}
