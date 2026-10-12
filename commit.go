package boxmodel

import "encoding/json"

func copyCommit(c Commit) Commit {
	b, _ := json.Marshal(c)
	var result Commit
	_ = json.Unmarshal(b, &result)
	return result
}
