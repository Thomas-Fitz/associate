package graph

import (
"context"
"testing"
)

func TestNewClient(t *testing.T) {
// Test client creation (won't connect without Neo4j)
client, err := NewClient("neo4j://localhost:7687", "neo4j", "password", "neo4j")
if err != nil {
t.Fatalf("NewClient() error = %v", err)
}
defer client.Close(context.Background())

if client == nil {
t.Error("NewClient() returned nil client")
}
}

func TestRepoNodeValidation(t *testing.T) {
tests := []struct {
name    string
node    *RepoNode
wantErr bool
}{
{
name: "valid repo node",
node: &RepoNode{
Path: "/path/to/repo",
Name: "my-repo",
},
wantErr: false,
},
{
name: "missing path",
node: &RepoNode{
Name: "my-repo",
},
wantErr: true,
},
{
name: "missing name",
node: &RepoNode{
Path: "/path/to/repo",
},
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := tt.node.Validate()
if (err != nil) != tt.wantErr {
t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}

func TestCodeNodeValidation(t *testing.T) {
tests := []struct {
name    string
node    *CodeNode
wantErr bool
}{
{
name: "valid function node",
node: &CodeNode{
Type:        "function",
Name:        "MyFunction",
FilePath:    "/path/to/file.go",
Description: "Does something",
},
wantErr: false,
},
{
name: "missing type",
node: &CodeNode{
Name:        "MyFunction",
FilePath:    "/path/to/file.go",
Description: "Does something",
},
wantErr: true,
},
{
name: "missing name",
node: &CodeNode{
Type:        "function",
FilePath:    "/path/to/file.go",
Description: "Does something",
},
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := tt.node.Validate()
if (err != nil) != tt.wantErr {
t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}

func TestMemoryNodeValidation(t *testing.T) {
tests := []struct {
name    string
node    *MemoryNode
wantErr bool
}{
{
name: "valid memory node",
node: &MemoryNode{
Content:     "This is a memory about the system",
ContextType: "architectural_decision",
Tags:        []string{"auth", "security"},
},
wantErr: false,
},
{
name: "missing content",
node: &MemoryNode{
ContextType: "architectural_decision",
},
wantErr: true,
},
{
name: "missing context type",
node: &MemoryNode{
Content: "Some content",
},
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := tt.node.Validate()
if (err != nil) != tt.wantErr {
t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}

func TestLearningNodeValidation(t *testing.T) {
tests := []struct {
name    string
node    *LearningNode
wantErr bool
}{
{
name: "valid learning node",
node: &LearningNode{
Pattern:     "Use service objects for complex business logic",
Category:    "architectural_pattern",
Description: "Keeps controllers thin",
},
wantErr: false,
},
{
name: "missing pattern",
node: &LearningNode{
Category:    "architectural_pattern",
Description: "Keeps controllers thin",
},
wantErr: true,
},
{
name: "missing category",
node: &LearningNode{
Pattern:     "Use service objects",
Description: "Keeps controllers thin",
},
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := tt.node.Validate()
if (err != nil) != tt.wantErr {
t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}
