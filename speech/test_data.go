package speech

// silentWAV is a minimal WAV file containing one silent sample
// This is used for testing audio playback functionality
var silentWAV = []byte{
	'R', 'I', 'F', 'F',
	0x26, 0x00, 0x00, 0x00, // RIFF chunk size = 36 + data(2) = 38
	'W', 'A', 'V', 'E',
	'f', 'm', 't', ' ',
	0x10, 0x00, 0x00, 0x00, // fmt chunk size = 16
	0x01, 0x00, // AudioFormat = 1 (PCM)
	0x01, 0x00, // NumChannels = 1 (mono)
	0x44, 0xAC, 0x00, 0x00, // SampleRate = 44100
	0x88, 0x58, 0x01, 0x00, // ByteRate = 44100 * 1 * 2 = 88200
	0x02, 0x00, // BlockAlign = 2
	0x10, 0x00, // BitsPerSample = 16
	'd', 'a', 't', 'a',
	0x02, 0x00, 0x00, 0x00, // Data size = 2 bytes (1 sample)
	0x00, 0x00, // Silent sample (16bit PCM 0)
}

// GetSilentWAV returns a copy of the silent WAV data for testing
func GetSilentWAV() []byte {
	result := make([]byte, len(silentWAV))
	copy(result, silentWAV)
	return result
}
