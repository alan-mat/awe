package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	pb "github.com/alan-mat/awe/internal/proto"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const gap = "\n\n"

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewAWEServiceClient(conn)

	p := tea.NewProgram(initialModel(client), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type (
	errMsg error
)

type responseMsg struct {
	Content string
	Done    bool
}

type model struct {
	client  pb.AWEServiceClient
	history []*pb.ChatMessage
	sub     chan responseMsg

	lastQuery  string
	acc        string
	viewport   viewport.Model
	entries    []string
	textarea   textarea.Model
	userStyle  lipgloss.Style
	modelStyle lipgloss.Style
	err        error
}

func initialModel(c pb.AWEServiceClient) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.ShowLineNumbers = false

	vp := viewport.New(30, 5)
	vp.SetContent(`Welcome to the chat room!
Type a message and press Enter to send.`)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		client:     c,
		sub:        make(chan responseMsg),
		textarea:   ta,
		entries:    []string{},
		viewport:   vp,
		userStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("34")),
		modelStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("31")),
		err:        nil,
	}
}

func sendChatMessage(m model, q string) tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	req := &pb.ChatRequest{
		User:    "tui-client",
		Query:   q,
		History: m.history,
	}

	stream, err := m.client.Chat(ctx, req)
	if err != nil {
		log.Fatalf("client.Chat failed: %v", err)
	}

	return func() tea.Msg {
		defer cancel()
		for {
			chatresp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("client.Chat failed: %v", err)
			}
			rmsg := responseMsg{
				Content: chatresp.Content,
				Done:    false,
			}
			m.sub <- rmsg
		}
		return responseMsg{Done: true}
	}
}

func waitForActivity(sub chan responseMsg) tea.Cmd {
	return func() tea.Msg {
		return responseMsg(<-sub)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		waitForActivity(m.sub),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.viewport.Height = msg.Height - m.textarea.Height() - lipgloss.Height(gap)

		if len(m.entries) > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.entries, "\n")))
		}
		m.viewport.GotoBottom()
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyEnter:
			q := m.textarea.Value()
			m.lastQuery = q
			m.entries = append(m.entries, m.userStyle.Render("You: ")+q)

			m.entries = append(m.entries, m.modelStyle.Render("Assistant: "))

			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.entries, "\n")))
			m.textarea.Reset()
			m.viewport.GotoBottom()
			m.textarea.Blur()
			return m, sendChatMessage(m, q)
		}

	case responseMsg:
		m.acc += msg.Content
		m.entries[len(m.entries)-1] = m.modelStyle.Render("Assistant: ") + strings.TrimSpace(m.acc)
		m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width).Render(strings.Join(m.entries, "\n")))
		m.viewport.GotoBottom()
		if msg.Done {
			m.textarea.Focus()
			m.history = append(m.history,
				&pb.ChatMessage{
					Role:    pb.ChatRole_USER,
					Content: m.lastQuery,
				},
				&pb.ChatMessage{
					Role:    pb.ChatRole_ASSISTANT,
					Content: strings.TrimSpace(m.acc),
				})
			m.acc = ""
		}
		return m, waitForActivity(m.sub)

	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s%s%s",
		m.viewport.View(),
		gap,
		m.textarea.View(),
	)
}
