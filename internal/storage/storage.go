package storage

import (
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go"
	"github.com/pkg/errors"
)

const (
	bucketNamePrefix = "demos-hub-server"
)

type Storage struct {
	uploadDir string
	client    *minio.Client
}

func NewStorage(client *minio.Client) *Storage {
	return &Storage{
		client: client,
	}
}

func (s *Storage) SaveFile(serverID, filename string, file io.Reader) error {
	name := bucketName(serverID)
	exists, err := s.client.BucketExists(name)
	if err != nil {
		return errors.Wrap(err, "failed to check if bucket exists")
	}

	if !exists {
		err = s.client.MakeBucket(name, "")
		if err != nil {
			return errors.Wrap(err, "failed to create bucket")
		}
	}

	_, err = s.client.PutObject(name, filename, file, -1, minio.PutObjectOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to upload file")
	}

	return nil
}

func (s *Storage) ListServers() ([]string, error) {
	buckets, err := s.client.ListBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list buckets")
	}

	var servers []string
	for _, bucket := range buckets {
		if strings.HasPrefix(bucket.Name, bucketNamePrefix) {
			servers = append(servers, strings.TrimPrefix(bucket.Name, bucketNamePrefix+"-"))
		}
	}

	return servers, nil
}

func (s *Storage) ListServerFiles(serverID string) ([]string, error) {
	objectsCh := s.client.ListObjects(bucketName(serverID), "", true, nil)

	var serverFiles []string
	for object := range objectsCh {
		if object.Err != nil {
			return nil, errors.Wrap(object.Err, "failed to list objects")
		}

		serverFiles = append(serverFiles, object.Key)
	}

	return serverFiles, nil
}

func bucketName(serverID string) string {
	return fmt.Sprintf("%s-%s", bucketNamePrefix, serverID)
}
