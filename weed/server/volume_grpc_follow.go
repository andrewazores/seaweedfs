package weed_server

import (
	"fmt"
	"github.com/chrislusf/seaweedfs/weed/storage/types"
	"io"
	"os"

	"github.com/chrislusf/seaweedfs/weed/pb/volume_server_pb"
	"github.com/chrislusf/seaweedfs/weed/storage"
)

func (vs *VolumeServer) VolumeFollow(req *volume_server_pb.VolumeFollowRequest, stream volume_server_pb.VolumeServer_VolumeFollowServer) error {

	v := vs.store.GetVolume(storage.VolumeId(req.VolumeId))
	if v == nil {
		return fmt.Errorf("not found volume id %d", req.VolumeId)
	}

	stopOffset := v.Size()
	foundOffset, isLastOne, err := v.BinarySearchByAppendAtNs(req.Since)
	if err != nil {
		return fmt.Errorf("fail to locate by appendAtNs: %s", err)
	}

	if isLastOne {
		return nil
	}

	startOffset := int64(foundOffset) * int64(types.NeedleEntrySize)

	buf := make([]byte, 1024*1024*2)
	return sendFileContent(v.DataFile(), buf, startOffset, stopOffset, stream)

}

func sendFileContent(datFile *os.File, buf []byte, startOffset, stopOffset int64, stream volume_server_pb.VolumeServer_VolumeFollowServer) error {
	var blockSizeLimit = int64(len(buf))
	for i := int64(0); i < stopOffset-startOffset; i += blockSizeLimit {
		n, readErr := datFile.ReadAt(buf, startOffset+i)
		if readErr == nil || readErr == io.EOF {
			resp := &volume_server_pb.VolumeFollowResponse{}
			resp.FileContent = buf[i : i+int64(n)]
			sendErr := stream.Send(resp)
			if sendErr != nil {
				return sendErr
			}
		} else {
			return readErr
		}
	}
	return nil
}
