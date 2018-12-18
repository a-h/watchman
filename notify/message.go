package notify

// NewMessage creates a notification message.
func NewMessage(subject, text string) Message {
	return Message{
		Text: subject,
		Attachments: []Attachment{
			Attachment{
				Text:  text,
				Color: "warning",
				Title: "Watchman",
			},
		},
	}
}

// A Message to send.
type Message struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

// An Attachment on a message.
type Attachment struct {
	Text  string `json:"text"`
	Color string `json:"color"`
	Title string `json:"title"`
}
