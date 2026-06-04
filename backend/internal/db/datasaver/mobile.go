package datasaver

import (
	"openreplay/backend/pkg/messages"
	sdk "openreplay/backend/pkg/sdk/model"
	"openreplay/backend/pkg/sessions"
)

func (s *saverImpl) handleMobileMessage(session *sessions.Session, msg messages.Message) error {
	switch m := msg.(type) {
	case *messages.MobileSessionEnd:
		if s.ch != nil {
			return s.ch.InsertMobileSession(session)
		}
	case *messages.MobileUserID:
		if s.users != nil {
			return s.users.Add(session, sdk.NewUser(m.ID))
		}
	case *messages.MobileUserAnonymousID:
		return s.sessions.UpdateAnonymousID(session.SessionID, m.ID)
	case *messages.MobileMetadata:
		return s.sessions.UpdateMetadata(m.SessionID(), m.Key, m.Value)
	case *messages.MobileEvent:
		if s.ch != nil {
			return s.ch.InsertMobileCustom(session, m)
		}
	case *messages.MobileClickEvent:
		if s.ch != nil {
			if err := s.ch.InsertMobileClick(session, m); err != nil {
				return err
			}
		}
		return s.sessions.UpdateEventsStats(session.SessionID, 1, 0)
	case *messages.MobileSwipeEvent:
		if s.ch != nil {
			if err := s.ch.InsertMobileSwipe(session, m); err != nil {
				return err
			}
		}
		return s.sessions.UpdateEventsStats(session.SessionID, 1, 0)
	case *messages.MobileInputEvent:
		if s.ch != nil {
			if err := s.ch.InsertMobileInput(session, m); err != nil {
				return err
			}
		}
		return s.sessions.UpdateEventsStats(session.SessionID, 1, 0)
	case *messages.MobileNetworkCall:
		if s.ch != nil {
			return s.ch.InsertMobileRequest(session, m, session.SaveRequestPayload)
		}
	case *messages.MobileCrash:
		if s.ch != nil {
			return s.ch.InsertMobileCrash(session, m)
		}
	}
	return nil
}
