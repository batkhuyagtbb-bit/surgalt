package store

import (
	"context"
	"sort"
)

// payoutMem — багшийн данс ба татах хүсэлт (payouts.go).
type payoutMem struct {
	accounts map[string]*PayoutAccount // багш → данс
	payouts  map[string]*Payout
}

func (p *payoutMem) init() {
	if p.accounts == nil {
		p.accounts, p.payouts = map[string]*PayoutAccount{}, map[string]*Payout{}
	}
}

func (m *Memory) PayoutAccount(_ context.Context, teacherID string) (*PayoutAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.pay.accounts[teacherID]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (m *Memory) SetPayoutAccount(_ context.Context, a *PayoutAccount) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pay.init()
	cp := *a
	m.pay.accounts[a.TeacherID] = &cp
	return nil
}

func (m *Memory) PayoutAccounts(_ context.Context) ([]PayoutAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []PayoutAccount{}
	for _, a := range m.pay.accounts {
		out = append(out, *a)
	}
	return out, nil
}

func (m *Memory) teacherPaidOrders(teacherID string) []Order {
	out := []Order{}
	for _, o := range m.orders {
		if earning(o, teacherID) {
			out = append(out, *o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PaidAt.After(*out[j].PaidAt) })
	return out
}

func (m *Memory) payoutsWhere(keep func(*Payout) bool) []Payout {
	out := []Payout{}
	for _, p := range m.pay.payouts {
		if keep(p) {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (m *Memory) TeacherPaidOrders(_ context.Context, teacherID string) ([]Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.teacherPaidOrders(teacherID), nil
}

func (m *Memory) teacherBalance(teacherID string) *Balance {
	return balanceOf(m.teacherPaidOrders(teacherID), m.payoutsWhere(func(p *Payout) bool { return p.TeacherID == teacherID }))
}

func (m *Memory) TeacherBalance(_ context.Context, teacherID string) (*Balance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.teacherBalance(teacherID), nil
}

func (m *Memory) RequestPayout(_ context.Context, teacherID string, amount int64, auto bool) (*Payout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.pay.accounts[teacherID]
	if !ok {
		return nil, ErrNotFound
	}
	p, err := newPayout(a, m.teacherBalance(teacherID), amount, auto)
	if err != nil {
		return nil, err
	}
	cp := *p
	m.pay.payouts[p.ID] = &cp
	return p, nil
}

func (m *Memory) Payouts(_ context.Context, teacherID, status string, limit int) ([]Payout, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := m.payoutsWhere(func(p *Payout) bool {
		return (teacherID == "" || p.TeacherID == teacherID) && (status == "" || p.Status == status)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) UpdatePayout(_ context.Context, id string, fn func(*Payout) error) (*Payout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.pay.payouts[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *p
	if err := fn(&cp); err != nil {
		return nil, err
	}
	*p = cp
	return &cp, nil
}
