package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/ethersphere/bee/v2/pkg/cac"
	"github.com/ethersphere/bee/v2/pkg/crypto"
	"github.com/ethersphere/bee/v2/pkg/soc"
	"github.com/ethersphere/bee/v2/pkg/swarm"
)

// FeedService represents Bee's Feed service
type FeedService service

// FeedUploadResponse represents a feed upload response
type FeedUploadResponse struct {
	Reference swarm.Address `json:"reference"`
	Owner     string
	Topic     string
}

// FindFeedUpdateResponse represents a feed update response
type FindFeedUpdateResponse struct {
	SocSignature string
	Index        uint64
	NextIndex    uint64
	Data         []byte
}

func ownerFromSigner(signer crypto.Signer) (string, error) {
	publicKey, err := signer.PublicKey()
	if err != nil {
		return "", err
	}
	ownerBytes, err := crypto.NewEthereumAddress(*publicKey)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ownerBytes), nil
}

// CreateRootManifest creates an initial feed root manifest
func (f *FeedService) CreateRootManifest(ctx context.Context, signer crypto.Signer, topic []byte, o UploadOptions) (*FeedUploadResponse, error) {
	ownerHex, err := ownerFromSigner(signer)
	if err != nil {
		return nil, err
	}
	topicHex := hex.EncodeToString(topic)
	h := http.Header{}
	if o.Pin {
		h.Add(swarmPinHeader, "true")
	}
	h.Add(postageStampBatchHeader, o.BatchID)
	var response FeedUploadResponse
	err = f.client.requestWithHeader(ctx, http.MethodPost, "/"+apiVersion+"/feeds/"+ownerHex+"/"+topicHex, h, nil, &response)
	if err != nil {
		return nil, err
	}

	response.Owner = ownerHex
	response.Topic = topicHex
	return &response, nil
}

// UpdateWithReference updates a feed with a reference. This is a type v1 feed update.
func (f *FeedService) UpdateWithReference(ctx context.Context, signer crypto.Signer, topic []byte, i uint64, addr swarm.Address, o UploadOptions) (*SocResponse, error) {
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(time.Now().Unix()))
	ch, err := cac.New(append(append([]byte{}, ts...), addr.Bytes()...))
	if err != nil {
		return nil, err
	}
	return f.UpdateWithRootChunk(ctx, signer, topic, i, ch, o)
}

// UpdateWithRootChunk updates a feed with a root chunk.
func (f *FeedService) UpdateWithRootChunk(ctx context.Context, signer crypto.Signer, topic []byte, i uint64, ch swarm.Chunk, o UploadOptions) (*SocResponse, error) {
	ownerHex, err := ownerFromSigner(signer)
	if err != nil {
		return nil, err
	}
	index := make([]byte, 8)
	binary.BigEndian.PutUint64(index, i)
	idBytes, err := crypto.LegacyKeccak256(slices.Concat(topic, index))
	if err != nil {
		return nil, err
	}
	sch, err := soc.New(idBytes, ch).Sign(signer)
	if err != nil {
		return nil, err
	}
	chunkData := sch.Data()
	signatureBytes := chunkData[swarm.HashSize : swarm.HashSize+swarm.SocSignatureSize]
	id := hex.EncodeToString(idBytes)
	sig := hex.EncodeToString(signatureBytes)
	res, err := f.client.SOC.UploadSOC(ctx, ownerHex, id, sig, bytes.NewReader(ch.Data()), o.BatchID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// FindUpdate finds the latest update for a feed
func (f *FeedService) FindUpdate(ctx context.Context, signer crypto.Signer, topic []byte, o *DownloadOptions) (*FindFeedUpdateResponse, error) {
	ownerHex, err := ownerFromSigner(signer)
	if err != nil {
		return nil, err
	}
	topicHex := hex.EncodeToString(topic)
	res, header, err := f.client.requestDataGetHeader(ctx, http.MethodGet, "/"+apiVersion+"/feeds/"+ownerHex+"/"+topicHex, nil, o)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	b, err := io.ReadAll(res)
	if err != nil {
		return nil, err
	}
	index, err := strconv.ParseUint(header.Get(swarmFeedIndexHeader), 10, 64)
	if err != nil {
		return nil, err
	}
	nextIndex, err := strconv.ParseUint(header.Get(swarmFeedIndexNextHeader), 10, 64)
	if err != nil {
		return nil, err
	}
	return &FindFeedUpdateResponse{
		SocSignature: header.Get(swarmSocSignatureHeader),
		Index:        index,
		NextIndex:    nextIndex,
		Data:         b,
	}, nil
}
