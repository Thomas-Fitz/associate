package docker

import (
"testing"
)

func TestDockerAvailable(t *testing.T) {
available := IsDockerAvailable()
if !available {
t.Skip("Docker is not available on this system")
}
}

func TestContainerExists(t *testing.T) {
if !IsDockerAvailable() {
t.Skip("Docker not available")
}

tests := []struct {
name          string
containerName string
shouldExist   bool
}{
{
name:          "non-existent container",
containerName: "this-container-should-not-exist-12345",
shouldExist:   false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
exists, err := ContainerExists(tt.containerName)
if err != nil {
t.Fatalf("ContainerExists() error = %v", err)
}
if exists != tt.shouldExist {
t.Errorf("ContainerExists() = %v, want %v", exists, tt.shouldExist)
}
})
}
}

func TestContainerRunning(t *testing.T) {
if !IsDockerAvailable() {
t.Skip("Docker not available")
}

// Test with a container that doesn't exist
running, err := IsContainerRunning("this-container-should-not-exist-12345")
if err != nil {
t.Fatalf("IsContainerRunning() error = %v", err)
}
if running {
t.Error("IsContainerRunning() should return false for non-existent container")
}
}

func TestParseDockerConfig(t *testing.T) {
tests := []struct {
name    string
config  *ContainerConfig
wantErr bool
}{
{
name: "valid config",
config: &ContainerConfig{
Name:     "test-neo4j",
Image:    "neo4j:5.25-community",
URI:      "neo4j://localhost:7687",
Username: "neo4j",
Password: "testpass",
},
wantErr: false,
},
{
name: "missing name",
config: &ContainerConfig{
Image:    "neo4j:5.25-community",
URI:      "neo4j://localhost:7687",
Username: "neo4j",
Password: "testpass",
},
wantErr: true,
},
{
name: "missing image",
config: &ContainerConfig{
Name:     "test-neo4j",
URI:      "neo4j://localhost:7687",
Username: "neo4j",
Password: "testpass",
},
wantErr: true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := tt.config.Validate()
if (err != nil) != tt.wantErr {
t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}

func TestEnsureContainer(t *testing.T) {
if !IsDockerAvailable() {
t.Skip("Docker not available")
}

config := &ContainerConfig{
Name:     "test-associate-neo4j",
Image:    "neo4j:5.25-community",
URI:      "neo4j://localhost:7687",
Username: "neo4j",
Password: "testpassword",
}

// Clean up any existing test container
defer func() {
StopContainer(config.Name)
RemoveContainer(config.Name)
}()

// First call should create the container
created, err := EnsureContainer(config)
if err != nil {
t.Fatalf("EnsureContainer() error = %v", err)
}
if !created {
t.Error("EnsureContainer() should return true when creating a new container")
}

// Second call should not create (container already exists and running)
created, err = EnsureContainer(config)
if err != nil {
t.Fatalf("EnsureContainer() error on second call = %v", err)
}
if created {
t.Error("EnsureContainer() should return false when container already exists")
}

// Stop the container
if err := StopContainer(config.Name); err != nil {
t.Fatalf("Failed to stop container: %v", err)
}

// Third call should start the existing stopped container
created, err = EnsureContainer(config)
if err != nil {
t.Fatalf("EnsureContainer() error on third call = %v", err)
}
if created {
t.Error("EnsureContainer() should return false when starting existing container")
}

// Verify container is running
running, err := IsContainerRunning(config.Name)
if err != nil {
t.Fatalf("Failed to check if container is running: %v", err)
}
if !running {
t.Error("Container should be running after EnsureContainer()")
}
}
