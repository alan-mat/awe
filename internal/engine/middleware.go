package engine

// TODO: handle MessageStream.Send errors
func TransportGenerationMiddleware(t Transport, streamId string) Middleware {
	return func(next Executer) Executer {
		return ExecuterFunc(func(c Context, p *Params) *Response {
			resp := next.Execute(c, p)
			if resp.GenerationChannel == nil {
				return resp
			}

			ms := t.MessageStream(streamId)

			ms.Send(c.Context(), MessageStreamPayload{
				Type:   TransportMessageStatus,
				Status: TransportStatusContentStart,
			})

			for evt := range resp.GenerationChannel {
				var payload *MessageStreamPayload
				switch evt.Type {
				case GenerationEventContentDelta:
					payload = &MessageStreamPayload{
						Type:    TransportMessageContent,
						Content: evt.Content,
					}
				case GenerationEventError:
					payload = &MessageStreamPayload{
						Type:  TransportMessageError,
						Error: evt.Error,
					}
				}

				if payload != nil {
					ms.Send(c.Context(), *payload)
				}
			}

			ms.Send(c.Context(), MessageStreamPayload{
				Type:   TransportMessageStatus,
				Status: TransportStatusContentEnd,
			})

			return resp
		})
	}
}
