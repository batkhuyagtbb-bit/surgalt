package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"surgalt/internal/khanbank"
	"surgalt/internal/store"
)

// ---- Хаан банк (Corporate Gateway): багшийн татах хүсэлтийг платформын данснаас шууд шилжүүлэх ----
// Урсгал: pending → sending (давхар илгээхгүй) → paid (журналын дугаартай) | pending (банк татгалзсан — шалтгаан
// тэмдэглэгдэнэ) | review (хариу тодорхойгүй — хянагч хуулгаас шалгаад «шилжүүлсэн» эсвэл «дахин илгээх»).
// Банкны үлдэгдэл, шилжүүлгийн удирдлага зөвхөн хянагчид харагдана.

// BankConfig — KHAN_CGW_* тохиргоо.
type BankConfig struct {
	FromAccount  string // платформын Хаан банкны данс (хоосон бол гэрээнд байгаа анхны MNT харилцах данс)
	Login        string // loginName (хоосон бол токен олгосон и-мэйл)
	TranPassword string // корпорэйт порталаас үүсгэсэн гүйлгээний нууц үг
	Auto         bool   // татах хүсэлтийг хянагч батлахгүйгээр шууд шилжүүлэх
	AutoMax      int64  // автоматаар шилжүүлэх дээд дүн (₮); түүнээс их бол хянагч батална
}

// bankInfo — Монголбанкны оролцогчийн RTGS код, BIC (payoutBanks-ийн түлхүүрээр).
var bankInfo = map[string]struct{ Code, BIC string }{
	"khan": {"050000", "AGMOMNUB"}, "golomt": {"150000", "GLMTMNUB"}, "tdb": {"040000", "TDBMMNUB"},
	"xac": {"320000", "CAXBMNUB"}, "state": {"340000", "STBMMNUB"}, "capitron": {"300000", "CPITMNUB"},
	"bogd": {"380000", "BOGDMNUB"}, "nibank": {"290000", "NAIMMNUB"}, "arig": {"210000", "ARGBMNUB"},
	"transport": {"190000", "TRDMMNUB"}, "ckb": {"330000", "CHKHMNUB"}, "mbank": {"390000", "MBNKMNUB"},
}

var (
	errBankOff = errors.New("Хаан банкны холболт тохируулаагүй")
	errBankPwd = errors.New("банкны гүйлгээний нууц үг (KHAN_CGW_TRAN_PASSWORD) тохируулаагүй")
	ibanRe     = regexp.MustCompile(`^MN[0-9]{2}([0-9]{4})([0-9]{12})$`)
)

// bankAccount — дансны дугаарыг шилжүүлэгт бэлдэнэ: Хаан банкны IBAN бол дотоод дансны дугаар руу хөрвүүлнэ.
func bankAccount(bank, acct string) string {
	acct = strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(acct))
	if m := ibanRe.FindStringSubmatch(acct); m != nil && bank == "khan" && m[1] == "0005" {
		return strings.TrimLeft(m[2], "0")
	}
	return acct
}

// bankFrom — шилжүүлэх данс: тохиргооны эсвэл гэрээнд байгаа анхны нээлттэй MNT харилцах данс (санана).
func (s *Server) bankFrom(ctx context.Context) (string, error) {
	if s.cfg.Bank.FromAccount != "" {
		return s.cfg.Bank.FromAccount, nil
	}
	s.khanFrom.Lock()
	defer s.khanFrom.Unlock()
	if s.khanFrom.acct != "" {
		return s.khanFrom.acct, nil
	}
	accts, err := s.Khan.Accounts(ctx)
	if err != nil {
		return "", err
	}
	for _, a := range accts {
		if a.Type == "D" && a.Currency == "MNT" && a.Status == "OPEN" {
			s.khanFrom.acct = a.Number
			return a.Number, nil
		}
	}
	return "", errors.New("Хаан банкны гэрээнд MNT харилцах данс олдсонгүй")
}

// bankHolder — ДНС-ээс данс эзэмшигчийн (далдалсан) нэр; банк холбоогүй бол nil.
func (s *Server) bankHolder(ctx context.Context, bank, acct string) (*khanbank.Holder, error) {
	bi, ok := bankInfo[bank]
	if s.Khan == nil || !ok {
		return nil, nil
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	a := bankAccount(bank, acct)
	if ibanRe.MatchString(a) {
		a = a[len(a)-12:] // IBAN-ы дансны хэсэг
		a = strings.TrimLeft(a, "0")
	}
	return s.Khan.NameInquiry(cctx, a, bi.Code)
}

// bankSend — хүлээгдэж буй татах хүсэлтийг Хаан банкаар шилжүүлнэ. by — хэн илгээсэн (хянагч эсвэл автомат).
func (s *Server) bankSend(ctx context.Context, id, by string) (*store.Payout, error) {
	if s.Khan == nil {
		return nil, errBankOff
	}
	if s.cfg.Bank.TranPassword == "" {
		return nil, errBankPwd
	}
	ctx = context.WithoutCancel(ctx) // хэрэглэгч хуудсаа хаасан ч эхэлсэн шилжүүлэг дуусна
	from, err := s.bankFrom(ctx)
	if err != nil {
		return nil, err
	}
	login := s.cfg.Bank.Login
	if login == "" {
		if login, err = s.Khan.DeveloperEmail(ctx); err != nil {
			return nil, err
		}
	}
	errState := errors.New("энэ хүсэлт хүлээгдэж буй төлөвт биш")
	ref := "SG" + strings.ToUpper(fmt.Sprintf("%x", time.Now().UnixNano()))[:14]
	p, err := s.store.UpdatePayout(ctx, id, func(p *store.Payout) error {
		if p.Status != store.PayoutPending {
			return errState
		}
		if _, ok := bankInfo[p.Bank]; !ok {
			return fmt.Errorf("банк %q-ийн кодыг мэдэхгүй", p.Bank)
		}
		p.Status, p.BankRef = store.PayoutSending, ref
		return nil
	})
	if err != nil {
		return nil, err
	}
	bi := bankInfo[p.Bank]
	t := khanbank.Transfer{FromAccount: from, ToAccount: bankAccount(p.Bank, p.AccountNo), Amount: p.Amount,
		Description: "surgalt.mn орлого " + p.ID[len(p.ID)-6:], TransferID: ref, LoginName: login, TranPassword: s.cfg.Bank.TranPassword}
	tctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	var res *khanbank.TransferResult
	if p.Bank == "khan" {
		res, err = s.Khan.Domestic(tctx, t)
	} else {
		t.ToAccountName, t.ToBank, t.ToBankBic = p.AccountName, bi.Code, bi.BIC
		res, err = s.Khan.Interbank(tctx, t)
	}
	cancel()
	now := time.Now()
	final, uerr := s.store.UpdatePayout(ctx, id, func(x *store.Payout) error {
		if x.Status != store.PayoutSending || x.BankRef != ref {
			return errState
		}
		switch {
		case err == nil:
			x.Status, x.Journal, x.ProcessedAt, x.ProcessedBy = store.PayoutPaid, res.Journal, &now, by
			x.Note = "Хаан банкаар шилжүүлсэн"
			if res.Journal != "" {
				x.Note += " · журнал " + res.Journal
			}
			if s.Khan.Test() {
				x.Note += " (ТЕСТ орчин)"
			}
		case khanbank.Rejected(err):
			x.Status, x.Note = store.PayoutPending, "Банк татгалзсан: "+err.Error()
		default:
			x.Status, x.Note = store.PayoutReview, "Банкны хариу тодорхойгүй ("+err.Error()+") — хуулгаас шалгаад шийднэ үү"
		}
		return nil
	})
	if uerr != nil {
		s.log.Error("банкны шилжүүлгийн төлөв хадгалах", "payout", id, "ref", ref, "bank_err", err, "err", uerr)
		return nil, uerr
	}
	s.adminInvalidate()
	switch final.Status {
	case store.PayoutPaid:
		s.log.Info("Хаан банкаар шилжүүлэв", "payout", id, "amount", final.Amount, "journal", final.Journal, "by", by, "test", s.Khan.Test())
		s.notify(ctx, &store.Notification{UserID: final.TeacherID, Type: NotifPayout, Link: "/me#wallet",
			Title: "✅ Орлого данс руу шилжлээ · " + money(final.Amount), Body: bankName(final.Bank) + " " + final.AccountNo})
	default:
		s.log.Warn("Хаан банкны шилжүүлэг амжилтгүй", "payout", id, "status", final.Status, "err", err)
		s.notifyAdmins(ctx, &store.Notification{Type: NotifPayout, Link: s.adminPath() + "#payouts",
			Title: "⚠️ Банкны шилжүүлэг амжилтгүй · " + money(final.Amount), Body: final.Note})
	}
	return final, nil
}

// autoBank — KHAN_CGW_AUTO үед шинэ татах хүсэлтийг шууд шилжүүлнэ. Дүн их, эсвэл данс сүүлийн 24 цагт
// солигдсон бол (бүртгэл алдагдсан байж болзошгүй) хянагч батлах хүртэл хүлээнэ.
func (s *Server) autoBank(p *store.Payout) {
	if s.Khan == nil || !s.cfg.Bank.Auto || s.cfg.Bank.TranPassword == "" || p == nil {
		return
	}
	if lim := s.cfg.Bank.AutoMax; lim > 0 && p.Amount > lim {
		return
	}
	go func() {
		ctx := context.Background()
		if a, err := s.store.PayoutAccount(ctx, p.TeacherID); err != nil || time.Since(a.UpdatedAt) < 24*time.Hour {
			return
		}
		if _, err := s.bankSend(ctx, p.ID, "Хаан банк (автомат)"); err != nil {
			s.log.Warn("автомат банкны шилжүүлэг", "payout", p.ID, "err", err)
		}
	}()
}

// handleAdminBank: GET /api/admin/bank — Хаан банкны холболт, шилжүүлэх дансны үлдэгдэл (зөвхөн хянагч).
func (s *Server) handleAdminBank(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	out := map[string]any{"configured": s.Khan != nil}
	if s.Khan == nil {
		writeJSON(w, http.StatusOK, out)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	out["test"], out["tran_password"], out["auto"], out["auto_max"] = s.Khan.Test(), s.cfg.Bank.TranPassword != "", s.cfg.Bank.Auto, s.cfg.Bank.AutoMax
	from, err := s.bankFrom(ctx)
	if err != nil {
		out["error"] = err.Error()
		writeJSON(w, http.StatusOK, out)
		return
	}
	out["from_account"] = from
	if login := s.cfg.Bank.Login; login != "" {
		out["login"] = login
	} else if e, err := s.Khan.DeveloperEmail(ctx); err == nil {
		out["login"] = e
	}
	if accts, err := s.Khan.Accounts(ctx); err == nil {
		for _, a := range accts {
			if a.Number == from {
				out["balance"], out["available"], out["account_name"] = a.Balance, a.Available, a.Name
			}
		}
	} else {
		out["error"] = err.Error()
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminPayoutSend: POST /api/admin/payouts/{id}/send — хянагч хүлээгдэж буй хүсэлтийг Хаан банкаар шилжүүлнэ.
func (s *Server) handleAdminPayoutSend(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	p, err := s.bankSend(r.Context(), r.PathValue("id"), admin.DisplayName)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "олдсонгүй")
	case err != nil:
		writeErr(w, http.StatusConflict, err.Error())
	default:
		writeJSON(w, http.StatusOK, p)
	}
}

// handleAdminPayoutHolder: GET /api/admin/payouts/{id}/holder — ДНС-ээс хүлээн авагчийн (далдалсан) нэр, IBAN.
func (s *Server) handleAdminPayoutHolder(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	ps, err := s.store.Payouts(r.Context(), "", "", 2000)
	if s.storeErr(w, r, err) {
		return
	}
	i := slices.IndexFunc(ps, func(p store.Payout) bool { return p.ID == r.PathValue("id") })
	if i < 0 {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	if s.Khan == nil {
		writeErr(w, http.StatusConflict, errBankOff.Error())
		return
	}
	h, err := s.bankHolder(r.Context(), ps[i].Bank, ps[i].AccountNo)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if h == nil {
		writeErr(w, http.StatusConflict, "энэ банкны кодыг мэдэхгүй")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": h.Name, "iban": h.IBAN, "entered": ps[i].AccountName})
}
