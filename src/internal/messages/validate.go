package messages

import (
	"fmt"
)

// func ValidatePeers(peers []string) ([]string, error, ErrorCode) {
// 	var validPeers []string
// 	var invalid []string
// 	for _, peer := range peers {
// 		peer = strings.TrimSpace(peer)

// 		err, _ := ValidatePeerFormat(peer)
// 		if err == nil {
// 			validPeers = append(validPeers, peer)
// 		} else {
// 			invalid = append(invalid, peer)
// 		}
// 	}
// 	if len(invalid) > 0 {
// 		return validPeers, fmt.Errorf("some peers were invalid and ignored: %v", invalid), E_INVALID_FORMAT
// 	}
// 	return validPeers, nil, E_NONE
// }

// -- Message Type Validators --

func (h *HelloMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (e *ErrorMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (g *GetPeersMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (p *PeersMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (g *GetObjectMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (i *IHaveObjectMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (o *ObjectMessage) Validate() (error, ErrorCode) {

	if o.Object == nil {
		return fmt.Errorf("object could not get parsed"), E_INVALID_FORMAT
	}

	return o.Object.Validate()
}

func (g *GetMempoolMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (m *MempoolMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (g *GetChainTipMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}

func (c *ChainTipMessage) Validate() (error, ErrorCode) {
	return nil, E_NONE
}
