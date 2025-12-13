package core

var Gconfig Config

type Config struct {
	Debug       bool   `mapstructure:"debug"`
	Daemon      bool   `mapstructure:"daemon"`
	DisableAuth bool   `mapstructure:"disable_auth"`
	Plugins     string `mapstructure:"plugins"`
	LogPath     string `mapstructure:"log-path"`
	DataPath    string `mapstructure:"data-path"`
	ExtraPath   string `mapstructure:"extra-path"`
	PID         string `mapstructure:"pid"`
	HTTP        struct {
		Port          uint16 `mapstructure:"port"`
		RedirectHTTPS bool   `mapstructure:"redirect_https"`
	} `mapstructure:"http"`

	HTTPS struct {
		Disable bool   `mapstructure:"disable"`
		Port    uint16 `mapstructure:"port"`
	} `mapstructure:"https"`
	MQTT struct {
		Host string `mapstructure:"host"`
		Port uint16 `mapstructure:"port"`
	} `mapstructure:"mqtt"`

	Simulator struct {
		MaxSlaveId uint16 `mapstructure:"max_slave_id"`
		ModbusRTU  []struct {
			Name     string `mapstructure:"name"`
			Rate     uint   `mapstructure:"rate"`
			DataRate uint   `mapstructure:"datarate"`
			StopBits uint   `mapstructure:"stopbits"`
			Parity   uint   `mapstructure:"parity"`
		} `mapstructure:"modbus-rtu"`
		Devices []struct {
			Name            string `json:"name" mapstructure:"name"`
			SN              string `json:"sn" mapstructure:"sn"`
			DeviceType      string `son:"device_type" mapstructure:"device_type"`
			DevicePlugin    string `json:"device_plugin" mapstructure:"device_plugin"`
			SoftwareVersion string `son:"software_version" mapstructure:"software_version"`
			Model           string `json:"model" mapstructure:"model"`
			Disable         bool   `json:"disable" mapstructure:"disable"`
			URL             string `json:"url" mapstructure:"url"`
		} `mapstructure:"devices"`
	}
}

const certPEM = `-----BEGIN CERTIFICATE-----
MIIDiTCCAnGgAwIBAgIUT3WylQul5udB1uOKlNdycXkqbH0wDQYJKoZIhvcNAQEL
BQAwVDELMAkGA1UEBhMCQ04xEDAOBgNVBAgMB0JlaWppbmcxEDAOBgNVBAcMB0Jl
aWppbmcxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNTEx
MjkwMDQ4NDBaFw0zNTExMjcwMDQ4NDBaMFQxCzAJBgNVBAYTAkNOMRAwDgYDVQQI
DAdCZWlqaW5nMRAwDgYDVQQHDAdCZWlqaW5nMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDD
HI9bmXre4j521HiDKu3oFCGsEPZXm6jU8zVQ/VI//YIkGGLZyHqhd0G+2zYQkAMv
h2eEB5DFmqqlBRvwKEAhpJvLxroOS8Lb0MLNABMUcDHNXnlRx98tRVo7M9S7/mSA
/9Nt51fjrqqXAIEL+K2/gSZ5X3u2IQj+mmHO8n3NRqeyW+tfCB2+mkDz3qZu8S4S
ri+U5cwe8S+fB1d8ygZjpZLE5oXpxAFPNMvVengaIv2ejwSyvAUgZCzIGABO+Rs+
cRbgziW43kBz8YxJYn1nfOF7bFb7zkWl+cmPMvO+lBSrLQBhIApC7g5G22Ji2a4v
pBTYKE43Us70EN2aep1fAgMBAAGjUzBRMB0GA1UdDgQWBBRnNs0iK8nZPxL0EXX9
H4IUWSAyvTAfBgNVHSMEGDAWgBRnNs0iK8nZPxL0EXX9H4IUWSAyvTAPBgNVHRMB
Af8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQB2yI3hUEoedy5tIHsopEtLHh+5
GvQlGWyV2pXfSPOr0wqWZvcOCCnRlz1p9gUR5aOkuN7nBEfiA/2ajTwWbT9IFtP4
E/jEDIPIs4H3Y0hfB1D24bxGki3Dbk8iWcXP5OH2eCojqTOLdbUrjw1t3PZlFSbh
5fnRHvAj6OSbkzGS7Cq151bvjEi77LMPYeOH18siGRXPfsLRWZlsWuJCtRG9vLZi
YRaJcQ4OFf2Zm1626CBd+cYYEXlxhmU0amWETYovIJ5Tu14JCUVGeQdkMr4QWMw1
RYMUHDPMsbA4isCiFaSLnaoMfppctpAEggRRcjWCIIVKgRK5LoDFvrNGpO1d
-----END CERTIFICATE-----`

const keyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDDHI9bmXre4j52
1HiDKu3oFCGsEPZXm6jU8zVQ/VI//YIkGGLZyHqhd0G+2zYQkAMvh2eEB5DFmqql
BRvwKEAhpJvLxroOS8Lb0MLNABMUcDHNXnlRx98tRVo7M9S7/mSA/9Nt51fjrqqX
AIEL+K2/gSZ5X3u2IQj+mmHO8n3NRqeyW+tfCB2+mkDz3qZu8S4Sri+U5cwe8S+f
B1d8ygZjpZLE5oXpxAFPNMvVengaIv2ejwSyvAUgZCzIGABO+Rs+cRbgziW43kBz
8YxJYn1nfOF7bFb7zkWl+cmPMvO+lBSrLQBhIApC7g5G22Ji2a4vpBTYKE43Us70
EN2aep1fAgMBAAECggEAKPFe0el4o7nVQsleSqQhDWDgGgPrNcIn4RvyNb8a2evA
OgPWBn5v4V8tsDe+9iXKTVh8K/QMeLL2mS9jx/ciUg0BVncqxuI2DzuVDUC1QEY0
5TQsgDFRj2Xsw9yiCRseiwVkID16L4CRMqO78L+r8jJPWQvk4Xi4MvlBihRPutn5
Wuy+cB4xy/sr2muSmGyWkaZ7qqJC9MFUDafIAgtQQZi4obCiVUNxcbkFwvtDaxMO
5f+MPa4j9i2paxQVU79OcZ6AeBbS6lgMWCYRi+CJjGa04Gd71THqq9LWefdIvdQh
Iit6gejwQ06gqgr2/mWLHVr5PZVwMRq/4BkO13e9HQKBgQDr0b9I+5sLIwCVWuxE
z0O+3q7tFQkZQvNTwo53KVDm61QTjaVcS1oyz8rzbzsHfsAHJF2oQqn0cDblekgR
xVw2dQ0EclUeofGxL936/xNkun71RkaxK1iQvxWbV0Ma5OwVjDOpYpsMeCXFxw25
RaVj3qgXuns9NpdurmHx2GkQ2wKBgQDTzwAa5NMJbW1G76ytjbX1yJ3e3JCslfng
+wgMh7gJDNZA4GSYbN/K66seTqTPAXFQuIaE8tndTi+v9g/ienLZPzR46eInmN/o
SZtNkhYIu7rlGp+ip9/4wff9i2HXGSwdpMjMZlqVahSdHBAYnCgPZwNnqnv+IGy6
mnTQ4xu6zQKBgQCxqR2xgEz4gPBJlWx3Eqd5Pw8OclCehYAIVIU8ZRYcQqLe8FHq
TKKxsTa3W89fADDvsIgW4dJk472X+R4etU+Zf2nFNdXG9D7APM3B8TXNJ2vKoZ1U
kNFyi2Nd2solkt4CBdROAonJRSM/84z1TfEiYnbFGasLHPvNWPdVWrMdDQKBgQC5
R/2uiPa263tJL0Xdd/Zxb7HyDv2bi4JPtSigVWS+vfT6QZCd6beGucsxstfmoTtv
wksiJ5I/TjLW+SeCFV07/1c2YlnMC6Xqe+EX5S/TKe1eloCId9Ortnnp2DCZSdLW
h5yDeRHKXEZ1/ONzs74zYwiOeYsHjXOvdIe1ZsWODQKBgDNJxdPfg6s3jg5WaXZO
7We9QQKrWeyWl3563pUM7mzex+Khu2TGfn0sCV85EP5ZkgxWtlVPPda4FtvWxKUH
f+0EsQQz8xitR53iUzbjWzeTLUjiQYykcDVYoH/z/ac0/eqlPnWBv22SJGzr/+5I
PwXGWVNgYgZmskyb4gF25jQb
-----END PRIVATE KEY-----`
