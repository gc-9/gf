package types

type AttachmentItem struct {
	*Attachment
	Url string `json:"url"`
}
