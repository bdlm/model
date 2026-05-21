package model

import (
	"github.com/bdlm/cast/v2"
)

type ModelData interface {
	cast.Tslice | cast.Tmap
}
