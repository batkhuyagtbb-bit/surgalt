package store

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- багшийн данс ба татах хүсэлт (payouts.go) ----

const payoutAccountCols = "teacher_id, bank, account_no, account_name, schedule, schedule_at, updated_at"

func (c *ClickHouse) payoutAccountsWhere(ctx context.Context, where string, args ...any) ([]PayoutAccount, error) {
	out := []PayoutAccount{}
	err := c.query(ctx, "SELECT "+payoutAccountCols+" FROM payout_accounts FINAL WHERE "+where, args, func(r driver.Rows) error {
		var a PayoutAccount
		if err := r.Scan(&a.TeacherID, &a.Bank, &a.AccountNo, &a.AccountName, &a.Schedule, &a.ScheduleAt, &a.UpdatedAt); err != nil {
			return err
		}
		out = append(out, a)
		return nil
	})
	return out, err
}

func (c *ClickHouse) PayoutAccount(ctx context.Context, teacherID string) (*PayoutAccount, error) {
	as, err := c.payoutAccountsWhere(ctx, "teacher_id = ?", teacherID)
	if err != nil {
		return nil, err
	}
	if len(as) == 0 {
		return nil, ErrNotFound
	}
	return &as[0], nil
}

func (c *ClickHouse) SetPayoutAccount(ctx context.Context, a *PayoutAccount) error {
	return c.insert(ctx, "payout_accounts", []string{"teacher_id", "bank", "account_no", "account_name", "schedule", "schedule_at", "updated_at", "ver"},
		a.TeacherID, a.Bank, a.AccountNo, a.AccountName, a.Schedule, a.ScheduleAt.UTC(), a.UpdatedAt.UTC(), ver())
}

func (c *ClickHouse) PayoutAccounts(ctx context.Context) ([]PayoutAccount, error) {
	return c.payoutAccountsWhere(ctx, "1")
}

func (c *ClickHouse) TeacherPaidOrders(ctx context.Context, teacherID string) ([]Order, error) {
	rows, err := c.ordersWhere(ctx, "teacher_id = ? AND status = ? AND kind NOT IN (?, ?)", "ORDER BY paid_at DESC",
		teacherID, string(OrderPaid), OrderKindStorage, OrderKindPlan)
	if err != nil {
		return nil, err
	}
	out := make([]Order, len(rows))
	for i := range rows {
		out[i] = rows[i].Order
	}
	return out, nil
}

const payoutCols = "id, teacher_id, amount, bank, account_no, account_name, status, auto, note, created_at, processed_at, processed_by, bank_ref, journal"

func (c *ClickHouse) writePayout(ctx context.Context, p *Payout) error {
	return c.insert(ctx, "payouts", []string{"id", "teacher_id", "amount", "bank", "account_no", "account_name", "status", "auto", "note",
		"created_at", "processed_at", "processed_by", "bank_ref", "journal", "ver"},
		p.ID, p.TeacherID, p.Amount, p.Bank, p.AccountNo, p.AccountName, p.Status, p.Auto, p.Note,
		p.CreatedAt.UTC(), nullTime(p.ProcessedAt), p.ProcessedBy, p.BankRef, p.Journal, ver())
}

func (c *ClickHouse) payoutsWhere(ctx context.Context, where string, args ...any) ([]Payout, error) {
	out := []Payout{}
	err := c.query(ctx, "SELECT "+payoutCols+" FROM payouts FINAL WHERE "+where, args, func(r driver.Rows) error {
		var p Payout
		if err := r.Scan(&p.ID, &p.TeacherID, &p.Amount, &p.Bank, &p.AccountNo, &p.AccountName, &p.Status, &p.Auto, &p.Note,
			&p.CreatedAt, &p.ProcessedAt, &p.ProcessedBy, &p.BankRef, &p.Journal); err != nil {
			return err
		}
		out = append(out, p)
		return nil
	})
	return out, err
}

func (c *ClickHouse) TeacherBalance(ctx context.Context, teacherID string) (*Balance, error) {
	orders, err := c.TeacherPaidOrders(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	ps, err := c.payoutsWhere(ctx, "teacher_id = ?", teacherID)
	if err != nil {
		return nil, err
	}
	return balanceOf(orders, ps), nil
}

func (c *ClickHouse) RequestPayout(ctx context.Context, teacherID string, amount int64, auto bool) (*Payout, error) {
	unlock, err := c.lock(ctx, "payout:"+teacherID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	a, err := c.PayoutAccount(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	bal, err := c.TeacherBalance(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	p, err := newPayout(a, bal, amount, auto)
	if err != nil {
		return nil, err
	}
	return p, c.writePayout(ctx, p)
}

func (c *ClickHouse) Payouts(ctx context.Context, teacherID, status string, limit int) ([]Payout, error) {
	where, args := "1", []any{}
	if teacherID != "" {
		where += " AND teacher_id = ?"
		args = append(args, teacherID)
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	return c.payoutsWhere(ctx, where+" ORDER BY created_at DESC LIMIT ?", append(args, limit)...)
}

func (c *ClickHouse) UpdatePayout(ctx context.Context, id string, fn func(*Payout) error) (*Payout, error) {
	ps, err := c.payoutsWhere(ctx, "id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(ps) == 0 {
		return nil, ErrNotFound
	}
	// Багшийн түгжээ: татах хүсэлт үүсгэхтэй зэрэг үлдэгдэл буруу тооцогдохгүй.
	unlock, err := c.lock(ctx, "payout:"+ps[0].TeacherID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if ps, err = c.payoutsWhere(ctx, "id = ?", id); err != nil {
		return nil, err
	}
	p := ps[0]
	if err := fn(&p); err != nil {
		return nil, err
	}
	return &p, c.writePayout(ctx, &p)
}
