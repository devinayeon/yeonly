package room

import (
	"crypto/rand"
	"errors"
	"time"

	"yeonly-server/internal/model"
)

var ErrRoomMissing = errors.New("ruang tidak ditemukan atau sudah berakhir")
var ErrIdentity = errors.New("identitas tersimpan tidak valid; silakan masuk kembali")

// JoinRoom issues a credential once; reconnects retain the same identity and alias.
func (m *Manager) JoinRoom(roomID, token, hostKey string) (model.Participant, error) {
	r, ok := m.GetRoom(roomID)
	if !ok {
		return model.Participant{}, ErrRoomMissing
	}
	r.Lock()
	defer r.Unlock()
	if hostKey != "" {
		if hostKey != r.HostKey {
			return model.Participant{}, ErrIdentity
		}
		r.LastActiveAt = time.Now()
		return r.HostParticipant, nil
	}
	if token != "" {
		p, ok := r.Members[token]
		if !ok {
			return model.Participant{}, ErrIdentity
		}
		r.LastActiveAt = time.Now()
		return p, nil
	}
	p := model.Participant{ID: rand.Text(), Token: rand.Text(), Alias: model.GenerateAlias()}
	r.Members[p.Token] = p
	r.LastActiveAt = time.Now()
	return p, nil
}

// AuthenticateParticipant never accepts an arbitrary public ID as a credential.
func (m *Manager) AuthenticateParticipant(roomID, token string) (model.Participant, bool) {
	r, ok := m.GetRoom(roomID)
	if !ok || token == "" {
		return model.Participant{}, false
	}
	r.RLock()
	defer r.RUnlock()
	if token == r.HostParticipant.Token {
		return r.HostParticipant, true
	}
	p, ok := r.Members[token]
	return p, ok
}
