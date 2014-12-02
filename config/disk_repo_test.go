package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	fakesys "github.com/cloudfoundry/bosh-agent/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-agent/uuid/fakes"

	. "github.com/cloudfoundry/bosh-micro-cli/config"
)

var _ = Describe("DiskRepo", func() {
	var (
		configService     DeploymentConfigService
		repo              DiskRepo
		fs                *fakesys.FakeFileSystem
		fakeUUIDGenerator *fakeuuid.FakeGenerator
		cloudProperties   map[string]interface{}
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		configService = NewFileSystemDeploymentConfigService("/fake/path", fs, logger)
		fakeUUIDGenerator = &fakeuuid.FakeGenerator{}
		repo = NewDiskRepo(configService, fakeUUIDGenerator)
		cloudProperties = map[string]interface{}{
			"fake-cloud_property-key": "fake-cloud-property-value",
		}
	})

	Describe("Save", func() {
		It("saves the disk record using the config service", func() {
			fakeUUIDGenerator.GeneratedUuid = "fake-guid-1"
			record, err := repo.Save("fake-cid", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())
			Expect(record).To(Equal(DiskRecord{
				ID:              "fake-guid-1",
				CID:             "fake-cid",
				Size:            1024,
				CloudProperties: cloudProperties,
			}))

			deploymentConfig, err := configService.Load()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := DeploymentFile{
				Disks: []DiskRecord{
					{
						ID:              "fake-guid-1",
						CID:             "fake-cid",
						Size:            1024,
						CloudProperties: cloudProperties,
					},
				},
			}
			Expect(deploymentConfig).To(Equal(expectedConfig))
		})
	})

	Describe("Find", func() {
		It("finds existing disk records", func() {
			fakeUUIDGenerator.GeneratedUuid = "fake-guid-1"
			savedRecord, err := repo.Save("fake-cid", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())

			foundRecord, found, err := repo.Find("fake-cid")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(foundRecord).To(Equal(savedRecord))
		})

		It("when the disk is not in the records, returns not found", func() {
			fakeUUIDGenerator.GeneratedUuid = "fake-guid-2"
			_, err := repo.Save("other-cid", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())

			_, found, err := repo.Find("fake-cid")
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})

	Describe("UpdateCurrent", func() {
		Context("when a disk record exists with the same ID", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUuid = "fake-uuid-1"
				_, err := repo.Save("fake-cid", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())
			})

			It("saves the disk record as current stemcell", func() {
				err := repo.UpdateCurrent("fake-uuid-1")
				Expect(err).ToNot(HaveOccurred())

				deploymentConfig, err := configService.Load()
				Expect(err).ToNot(HaveOccurred())

				Expect(deploymentConfig.CurrentDiskID).To(Equal("fake-uuid-1"))
			})
		})

		Context("when a disk record does not exists with the same ID", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUuid = "fake-uuid-1"
				_, err := repo.Save("fake-cid", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := repo.UpdateCurrent("fake-uuid-2")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Verifying disk record exists with id `fake-uuid-2'"))
			})
		})
	})

	Describe("FindCurrent", func() {
		Context("when current disk exists", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUuid = "fake-guid-1"
				_, err := repo.Save("fake-cid-1", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())

				fakeUUIDGenerator.GeneratedUuid = "fake-guid-2"
				record, err := repo.Save("fake-cid-2", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())

				repo.UpdateCurrent(record.ID)
			})

			It("returns existing disk", func() {
				record, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(record).To(Equal(DiskRecord{
					ID:              "fake-guid-2",
					CID:             "fake-cid-2",
					Size:            1024,
					CloudProperties: cloudProperties,
				}))
			})
		})

		Context("when current disk does not exist", func() {
			BeforeEach(func() {
				fakeUUIDGenerator.GeneratedUuid = "fake-guid-1"
				_, err := repo.Save("fake-cid", 1024, cloudProperties)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns not found", func() {
				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})

		Context("when there are no disks", func() {
			It("returns not found", func() {
				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})

	Describe("All", func() {
		var (
			firstDisk  DiskRecord
			secondDisk DiskRecord
		)

		BeforeEach(func() {
			var err error
			fakeUUIDGenerator.GeneratedUuid = "fake-guid-1"
			firstDisk, err = repo.Save("fake-cid-1", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())

			fakeUUIDGenerator.GeneratedUuid = "fake-guid-2"
			secondDisk, err = repo.Save("fake-cid-2", 2048, cloudProperties)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns all disks", func() {
			disks, err := repo.All()
			Expect(err).ToNot(HaveOccurred())
			Expect(disks).To(Equal([]DiskRecord{
				firstDisk,
				secondDisk,
			}))
		})
	})

	Describe("Delete", func() {
		var (
			firstDisk  DiskRecord
			secondDisk DiskRecord
		)

		BeforeEach(func() {
			var err error

			fakeUUIDGenerator.GeneratedUuid = "fake-guid-1"
			firstDisk, err = repo.Save("fake-cid-1", 1024, cloudProperties)
			Expect(err).ToNot(HaveOccurred())

			fakeUUIDGenerator.GeneratedUuid = "fake-guid-2"
			secondDisk, err = repo.Save("fake-cid-2", 2048, cloudProperties)
			Expect(err).ToNot(HaveOccurred())
		})

		It("removes the disk record from the repo", func() {
			err := repo.Delete(firstDisk)
			Expect(err).ToNot(HaveOccurred())

			disks, err := repo.All()
			Expect(err).ToNot(HaveOccurred())
			Expect(disks).To(Equal([]DiskRecord{
				secondDisk,
			}))
		})
	})

	Describe("ClearCurrent", func() {
		It("updates disk cid", func() {
			err := repo.ClearCurrent()
			Expect(err).ToNot(HaveOccurred())

			deploymentConfig, err := configService.Load()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := DeploymentFile{
				CurrentDiskID: "",
			}
			Expect(deploymentConfig).To(Equal(expectedConfig))

			_, found, err := repo.FindCurrent()
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})
})
