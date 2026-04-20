package testcontainers 

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUDPPortBinding(t *testing.T) {
	ctx := context.Background()

	t.Run("UDP port gets proper host port allocation", func(t *testing.T) {
		// Create container with UDP port exposed
		req := ContainerRequest{
			Image:        "alpine/socat:latest",
			ExposedPorts: []string{"8080/udp","1234/tcp"},
			Cmd: []string{
        "sh", "-c",
        "socat UDP-LISTEN:8080,fork,reuseaddr EXEC:'/bin/cat' & socat TCP-LISTEN:8080,fork,reuseaddr EXEC:'/bin/cat'",
    	},
			// Cmd:          []string{"UDP-LISTEN:8080,fork,reuseaddr", "EXEC:'/bin/cat'"},
		}

		container, err := GenericContainer(ctx, GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, container.Terminate(ctx))
		}()

		// Test MappedPort function - this was the bug
		udpPort, err := nat.NewPort("udp", "8080")
		require.NoError(t, err)

		mappedPort, err := container.MappedPort(ctx, udpPort)
		require.NoError(t, err)
    t.Logf("UDP port is: %v", mappedPort.Port())
    
		mappedTcPPort, err := container.MappedPort(ctx, "1234/tcp")
		require.NoError(t, err)
		t.Logf("Tcp port is: %v", mappedTcPPort.Port())

		// Before fix: mappedPort.Port() would return "0"
		// After fix: mappedPort.Port() returns actual port like "55051"
		assert.NotEqual(t, "0", mappedPort.Port(), "UDP port should not return '0'")
		assert.Equal(t, "udp", mappedPort.Proto(), "Protocol should be UDP")

		portNum := mappedPort.Int()
		assert.Positive(t, portNum, "Port number should be greater than 0")
		assert.LessOrEqual(t, portNum, 65535, "Port number should be valid UDP port range")

		// Verify the port is actually accessible (basic connectivity test)
		hostIP, err := container.Host(ctx)
		require.NoError(t, err)

		address := net.JoinHostPort(hostIP, mappedPort.Port())
		conn, err := net.DialTimeout("udp", address, 2*time.Second)
		require.NoError(t, err, "Should be able to connect to UDP port")
		conn.Close()
	})
}
