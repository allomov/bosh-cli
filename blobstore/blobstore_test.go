package blobstore_test

import (
	"errors"
	"io/ioutil"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"

	fakeboshdavcli "github.com/cloudfoundry/bosh-agent/davcli/client/fakes"
	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-agent/uuid/fakes"

	. "github.com/cloudfoundry/bosh-micro-cli/blobstore"
)

var _ = Describe("Blobstore", func() {
	var (
		fakeDavClient     *fakeboshdavcli.FakeClient
		fakeUUIDGenerator *fakeuuid.FakeGenerator
		fs                *fakesys.FakeFileSystem
		blobstore         Blobstore
	)

	BeforeEach(func() {
		fakeDavClient = fakeboshdavcli.NewFakeClient()
		fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
		fs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)

		blobstore = NewBlobstore(fakeDavClient, fakeUUIDGenerator, fs, logger)
	})

	Describe("Get", func() {
		It("gets the blob from the blobstore", func() {
			fakeDavClient.GetContents = ioutil.NopCloser(strings.NewReader("fake-content"))

			err := blobstore.Get("fake-blob-id", "fake-destination-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeDavClient.GetPath).To(Equal("fake-blob-id"))
		})

		It("saves the blob to the destination path", func() {
			fakeDavClient.GetContents = ioutil.NopCloser(strings.NewReader("fake-content"))

			err := blobstore.Get("fake-blob-id", "fake-destination-path")
			Expect(err).ToNot(HaveOccurred())

			contents, err := fs.ReadFileString("fake-destination-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(contents).To(Equal("fake-content"))
		})

		Context("when getting from blobstore fails", func() {
			It("returns an error", func() {
				fakeDavClient.GetErr = errors.New("fake-get-error")
				err := blobstore.Get("fake-blob-id", "fake-destination-path")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-get-error"))
			})
		})
	})

	Describe("Add", func() {
		BeforeEach(func() {
			fs.RegisterOpenFile("fake-source-path", &fakesys.FakeFile{
				Contents: []byte("fake-contents"),
			})
		})

		It("adds file to blobstore and returns its blob ID", func() {
			fakeUUIDGenerator.GeneratedUuid = "fake-blob-id"

			blobID, err := blobstore.Add("fake-source-path")
			Expect(err).ToNot(HaveOccurred())
			Expect(blobID).To(Equal("fake-blob-id"))
			Expect(fakeDavClient.PutPath).To(Equal("fake-blob-id"))
			Expect(fakeDavClient.PutContents).To(Equal("fake-contents"))
		})
	})
})
