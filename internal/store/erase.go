package store

// ---- Бүртгэл устгах (Хувь хүний мэдээлэл хамгаалах тухай хууль — устгуулах эрх) ----
// DeleteUserData нь хэрэглэгчийн хувийн мэдээллийг устгана: суралцахуй (ахиц, шалгалт, даалгавар, тэмдэглэл),
// хичээлийн хяналт, мэдэгдэл, сэтгэгдэл, элсэлт ба хандалт, гадаад нэвтрэлтийн холбоос. Чатын мессежүүд нь
// «устгагдсан» болж нэр нь арилна; өнгөрсөн цаг захиалгад нэр нь арилна. Хэрэглэгчийн мөр нэргүй болж хаагдана
// (имэйлийг дахин бүртгүүлж болно). Төлбөрийн баримт (orders, payment_invoices) хуулийн дагуу, чатын мэдүүлэг
// аюулгүй байдлын үүднээс (2 жил) үлдэнэ.

// DeletedName — устгагдсан хэрэглэгчийн оронд харагдах нэр.
const DeletedName = "Устгагдсан хэрэглэгч"

// anonymize — хэрэглэгчийн мөрийг таних мэдээлэлгүй болгоно (зөвшөөрлийн түүх үлдэнэ).
func anonymize(u *User) {
	tail := u.ID
	if len(tail) > 8 {
		tail = tail[len(tail)-8:]
	}
	u.Email, u.Username, u.DisplayName = "deleted+"+u.ID+"@users.noreply.surgalt", "deleted_"+tail, DeletedName
	u.PasswordHash, u.Headline, u.Bio, u.AvatarURL, u.CoverURL = "", "", "", "", ""
	u.Subjects, u.Location, u.Links, u.Phone, u.GoogleToken, u.MeetConnected = nil, "", nil, "", "", false
}
