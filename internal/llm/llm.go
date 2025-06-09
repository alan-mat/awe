// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package llm

// Message contains a single message, consisting of many parts.
type Message struct {
	Role  MessageRole
	Parts []MessagePart
}

// Text returns all the text parts of a Message.
func (m Message) Text() string {
	out := ""
	for _, p := range m.Parts {
		if p.Type == MessagePartTypeText {
			t, _ := p.Text()
			out += t
		}
	}
	return out
}

// TextMessage is a helper function that returns a Message
// with a single text part.
func TextMessage(role MessageRole, text string) Message {
	return Message{
		Role: role,
		Parts: []MessagePart{
			NewTextPart(text),
		},
	}
}

// TextMessage is a helper function that returns a Message
// with a single blob part.
func BlobMessage(role MessageRole, mimeType string, b []byte) Message {
	return Message{
		Role: role,
		Parts: []MessagePart{
			NewBlobPart(mimeType, b),
		},
	}
}

// MessageRole defines the source of the message.
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

// MessagePart is a single message part that may hold
// a payload of different [MessagePartType]s.
type MessagePart struct {
	Type MessagePartType
	text *string
	blob *Blob
}

// NewTextPart creates a new [MessagePart] containing a text payload.
func NewTextPart(text string) MessagePart {
	return MessagePart{
		Type: MessagePartTypeText,
		text: &text,
	}
}

// NewBlobPart creates a new [MessagePart] containing a [Blob] payload.
func NewBlobPart(mimeType string, b []byte) MessagePart {
	return MessagePart{
		Type: MessagePartTypeBlob,
		blob: &Blob{
			MIMEType: mimeType,
			Data:     b,
		},
	}
}

// Text returns the text payload.
// Error returns not-nil if the part's type is not text,
// or if the text payload is nil.
func (p MessagePart) Text() (string, error) {
	if p.Type != MessagePartTypeText {
		return "", MismatchMessagePartTypeError{
			Wanted: MessagePartTypeText,
			Real:   p.Type,
		}
	}

	if p.text == nil {
		return "", NilPayloadError{Type: MessagePartTypeText}
	}

	return *p.text, nil
}

// Blob returns the blob (e.g. image) payload.
// Error returns not-nil if the part's type is not blob,
// or if the blob payload is empty. If this method returns a
// nil error then the returned value will always be a not-nil pointer to [Blob].
func (p MessagePart) Blob() (*Blob, error) {
	if p.Type != MessagePartTypeBlob {
		return nil, MismatchMessagePartTypeError{
			Wanted: MessagePartTypeBlob,
			Real:   p.Type,
		}
	}

	if p.blob == nil {
		return nil, NilPayloadError{Type: MessagePartTypeBlob}
	}

	return p.blob, nil
}

// MessagePartType specifies the payload type of a message part.
type MessagePartType string

const (
	MessagePartTypeText MessagePartType = "text"
	MessagePartTypeBlob MessagePartType = "blob"
)

// Blob contains media as raw bytes.
// The type of source data follows the IANA standard MIME type.
type Blob struct {
	MIMEType string
	Data     []byte
}
