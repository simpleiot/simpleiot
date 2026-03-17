package data

// MsgService is used to represent message services such as Twilio, SMTP, etc
type MsgService struct {
	ID        string
	Service   string
	SID       string
	AuthToken string
	From      string
}

// NodeToMsgService converts a node to message service
func NodeToMsgService(node Node) (MsgService, error) {
	ret := MsgService{}
	ret.ID = node.ID
	for _, p := range node.Points {
		switch p.Type {
		case PointTypeService:
			ret.Service = p.Txt()
		case PointTypeSID:
			ret.SID = p.Txt()
		case PointTypeAuthToken:
			ret.AuthToken = p.Txt()
		case PointTypeFrom:
			ret.From = p.Txt()
		}
	}

	return ret, nil
}
