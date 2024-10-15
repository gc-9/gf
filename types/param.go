package types

type IgnoreAutoValidate struct {
}

func (i *IgnoreAutoValidate) IgnoreAutoValidate() {
}

type ParamID struct {
	ID int `json:"id" form:"id" query:"id" validate:"required"`
}

type ParamUID struct {
	UID int `json:"uid" form:"uid" query:"uid" validate:"required"`
}

type PairValue[L string | int, R string | int] struct {
	Id   L `json:"id"`
	Name R `json:"name"`
}

type ParamToggleStatus struct {
	ID     int `json:"id" validate:"required" comment:"ID"`
	Status int `json:"status" validate:"required" comment:"状态"` // 状态
}

type ParamOptionsSearch struct {
	Q     string `json:"q" form:"q" query:"q"`
	Limit int    `json:"limit" form:"limit" query:"limit"`
}
