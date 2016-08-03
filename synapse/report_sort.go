package synapse

import (
	"sort"
	"github.com/n0rad/go-erlog/errs"
	"github.com/kubernetes/kubernetes/pkg/util/rand"
	"github.com/n0rad/go-erlog/data"
	"strings"
	"github.com/mitchellh/packer/common/json"
)

type ReportSortType string

func (n ReportSortType) Sort(reports *[]Report) {
	switch n {
	case SORT_RANDOM:
		for i := range *reports {
			j := rand.Intn(i + 1)
			(*reports)[i], (*reports)[j] = (*reports)[j], (*reports)[i]
		}
	case SORT_NAME:
		sort.Sort(ByName{*reports})
	case SORT_DATE:
		sort.Sort(ByDate{*reports})
	}
}

type Reports []Report

func (s Reports) Len() int {
	return len(s)
}
func (s Reports) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type ByName struct{ Reports }

func (s ByName) Less(i, j int) bool {
	return s.Reports[i].Name < s.Reports[j].Name
}

type ByDate struct{ Reports }

func (s ByDate) Less(i, j int) bool {
	return s.Reports[i].CreationTime < s.Reports[j].CreationTime
}

func (n *ReportSortType) UnmarshalJSON(d []byte) error {
	var s string
	if err := json.Unmarshal(d, &s); err != nil {
		return errs.WithF(data.WithField("value", string(d)), "Failed to unmarsal serverSort")
	}

	switch strings.ToLower(s) {
	case string(SORT_RANDOM):
		*n = SORT_RANDOM
	case string(SORT_NAME):
		*n = SORT_NAME
	case string(SORT_DATE):
		*n = SORT_DATE
	default:
		return errs.WithF(data.WithField("value", s), "Unknown serverSort")
	}
	return nil
}

const SORT_RANDOM ReportSortType = "random"
const SORT_NAME ReportSortType = "name"
const SORT_DATE ReportSortType = "date"
