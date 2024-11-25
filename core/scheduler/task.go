package scheduler

import (
	"encoding/json"
)

type Excutor struct {
	Mode    string `json:"mode"`
	Crontab string `json:"crontab"`
	Excute  string `json:"excute"`
}

type Task struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	StartedAt  int64  `json:"started_at"`
	FinishedAt int64  `json:"finished_at"`
	Excutor
}

func (t *Task) Marshal() string {
	b, _ := json.Marshal(t)
	return string(b)
}

func (t *Task) Unmarshal(data string) error {
	return json.Unmarshal([]byte(data), t)
}
