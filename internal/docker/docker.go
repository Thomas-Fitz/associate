// Package docker provides Docker container lifecycle management using the Docker CLI.
package docker

import (
"bytes"
"fmt"
"os/exec"
"strings"
"time"
)

// ContainerConfig holds the configuration for a Neo4j container.
type ContainerConfig struct {
Name     string
Image    string
URI      string
Username string
Password string
}

// Validate checks that all required fields are set.
func (c *ContainerConfig) Validate() error {
var missing []string

if c.Name == "" {
missing = append(missing, "Name")
}
if c.Image == "" {
missing = append(missing, "Image")
}
if c.URI == "" {
missing = append(missing, "URI")
}
if c.Username == "" {
missing = append(missing, "Username")
}
if c.Password == "" {
missing = append(missing, "Password")
}

if len(missing) > 0 {
return fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))
}

return nil
}

// IsDockerAvailable checks if Docker is installed and accessible.
func IsDockerAvailable() bool {
cmd := exec.Command("docker", "version")
return cmd.Run() == nil
}

// ContainerExists checks if a container with the given name exists.
func ContainerExists(name string) (bool, error) {
cmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Names}}")
output, err := cmd.Output()
if err != nil {
return false, fmt.Errorf("failed to check container existence: %w", err)
}

return strings.TrimSpace(string(output)) == name, nil
}

// IsContainerRunning checks if a container is currently running.
func IsContainerRunning(name string) (bool, error) {
cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=^%s$", name), "--format", "{{.Names}}")
output, err := cmd.Output()
if err != nil {
return false, fmt.Errorf("failed to check container status: %w", err)
}

return strings.TrimSpace(string(output)) == name, nil
}

// CreateContainer creates a new Neo4j container with the specified configuration.
func CreateContainer(config *ContainerConfig) error {
if err := config.Validate(); err != nil {
return fmt.Errorf("invalid container config: %w", err)
}

args := []string{
"run",
"-d",
"--name", config.Name,
"-p", "7687:7687",
"-p", "7474:7474",
"-e", fmt.Sprintf("NEO4J_AUTH=%s/%s", config.Username, config.Password),
config.Image,
}

cmd := exec.Command("docker", args...)
var stderr bytes.Buffer
cmd.Stderr = &stderr

if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to create container: %w (stderr: %s)", err, stderr.String())
}

return nil
}

// StartContainer starts an existing container.
func StartContainer(name string) error {
cmd := exec.Command("docker", "start", name)
var stderr bytes.Buffer
cmd.Stderr = &stderr

if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to start container: %w (stderr: %s)", err, stderr.String())
}

return nil
}

// StopContainer stops a running container.
func StopContainer(name string) error {
cmd := exec.Command("docker", "stop", name)
var stderr bytes.Buffer
cmd.Stderr = &stderr

if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to stop container: %w (stderr: %s)", err, stderr.String())
}

return nil
}

// RemoveContainer removes a container (must be stopped first).
func RemoveContainer(name string) error {
cmd := exec.Command("docker", "rm", name)
var stderr bytes.Buffer
cmd.Stderr = &stderr

if err := cmd.Run(); err != nil {
return fmt.Errorf("failed to remove container: %w (stderr: %s)", err, stderr.String())
}

return nil
}

// EnsureContainer ensures that a Neo4j container is running.
// If the container doesn't exist, it creates it.
// If the container exists but is stopped, it starts it.
// Returns true if the container was created, false if it already existed.
func EnsureContainer(config *ContainerConfig) (created bool, err error) {
if !IsDockerAvailable() {
return false, fmt.Errorf("Docker is not available. Please install Docker and ensure it is running")
}

if err := config.Validate(); err != nil {
return false, fmt.Errorf("invalid container config: %w", err)
}

exists, err := ContainerExists(config.Name)
if err != nil {
return false, err
}

if !exists {
// Container doesn't exist, create it
if err := CreateContainer(config); err != nil {
return false, err
}

// Wait a moment for container to initialize
time.Sleep(2 * time.Second)

return true, nil
}

// Container exists, check if it's running
running, err := IsContainerRunning(config.Name)
if err != nil {
return false, err
}

if !running {
// Container exists but not running, start it
if err := StartContainer(config.Name); err != nil {
return false, err
}

// Wait a moment for container to start
time.Sleep(2 * time.Second)
}

return false, nil
}

// WaitForContainer waits for a container to be ready by checking its health.
// For Neo4j, we check if the bolt port is accepting connections.
func WaitForContainer(name string, timeout time.Duration) error {
deadline := time.Now().Add(timeout)

for time.Now().Before(deadline) {
// Check if container is running
running, err := IsContainerRunning(name)
if err != nil {
return err
}

if !running {
return fmt.Errorf("container %s is not running", name)
}

// Check container logs for "Started." message which indicates Neo4j is ready
cmd := exec.Command("docker", "logs", name)
output, err := cmd.Output()
if err == nil && strings.Contains(string(output), "Started.") {
return nil
}

time.Sleep(1 * time.Second)
}

return fmt.Errorf("timeout waiting for container %s to be ready", name)
}
