package course

import "errors"

var ErrNotFound = errors.New("course not found")

type Course struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Hours       int    `json:"hours"`
	Level       string `json:"level"`
}

type Store interface {
	List() []Course
	Get(id string) (Course, error)
}

type MemoryStore struct {
	courses []Course
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		courses: []Course{
			{
				ID:          "go-basics",
				Title:       "Go Basics",
				Description: "Syntax, modules, packages and standard library basics.",
				Hours:       12,
				Level:       "beginner",
			},
			{
				ID:          "docker-kubernetes",
				Title:       "Docker and Kubernetes",
				Description: "Container image build, Kubernetes deployment and ingress setup.",
				Hours:       18,
				Level:       "intermediate",
			},
			{
				ID:          "cicd-github-actions",
				Title:       "CI/CD with GitHub Actions",
				Description: "Tag-based pipelines, image publishing and Kubernetes delivery.",
				Hours:       16,
				Level:       "intermediate",
			},
		},
	}
}

func (s *MemoryStore) List() []Course {
	result := make([]Course, len(s.courses))
	copy(result, s.courses)
	return result
}

func (s *MemoryStore) Get(id string) (Course, error) {
	for _, item := range s.courses {
		if item.ID == id {
			return item, nil
		}
	}

	return Course{}, ErrNotFound
}
