package topgun_test

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe(":life Garbage collecting resource cache volumes", func() {
	Describe("A resource that was removed from pipeline", func() {
		BeforeEach(func() {
			Deploy("deployments/single-vm.yml")
		})

		It("has its resource cache, resource cache uses and resource cache volumes cleared out", func() {
			By("setting pipeline that creates resource cache")
			fly("set-pipeline", "-n", "-c", "pipelines/get-task-changing-resource.yml", "-p", "volume-gc-test")

			By("unpausing the pipeline")
			fly("unpause-pipeline", "-p", "volume-gc-test")

			By("triggering the job")
			fly("trigger-job", "-w", "-j", "volume-gc-test/simple-job")

			By("getting the resource cache volumes")
			Expect(volumesByResourceType("time")).To(HaveLen(1))

			By("getting the resource caches")
			var resourceCachesNum int
			err := psql.Select("COUNT(id)").From("resource_caches").Where("version LIKE ?", fmt.Sprint("%", "time", "%")).RunWith(dbConn).QueryRow().Scan(&resourceCachesNum)
			Expect(err).ToNot(HaveOccurred())
			Expect(resourceCachesNum).To(Equal(1))

			By("getting the resource caches uses")
			var resourceCacheUsesNum int
			err = psql.Select("COUNT(*)").From("resource_cache_uses").RunWith(dbConn).QueryRow().Scan(&resourceCacheUsesNum)
			Expect(err).ToNot(HaveOccurred())
			// one for image resource, one for time
			Expect(resourceCacheUsesNum).To(Equal(2))

			By("updating pipeline and removing resource")
			fly("set-pipeline", "-n", "-c", "pipelines/task-waiting.yml", "-p", "volume-gc-test")

			By("eventually expiring the resource cache volumes")
			Eventually(func() int {
				return len(volumesByResourceType("time"))
			}, 5*time.Minute, 10*time.Second).Should(BeZero())

			By("expiring the resource caches")
			err = psql.Select("COUNT(id)").From("resource_caches").Where("version LIKE ?", fmt.Sprint("%", "time", "%")).RunWith(dbConn).QueryRow().Scan(&resourceCachesNum)
			Expect(err).ToNot(HaveOccurred())
			Expect(resourceCachesNum).To(BeZero())

			By("expiring the resource caches uses")
			err = psql.Select("COUNT(*)").From("resource_cache_uses").RunWith(dbConn).QueryRow().Scan(&resourceCacheUsesNum)
			Expect(err).ToNot(HaveOccurred())
			Expect(resourceCacheUsesNum).To(BeZero())
		})
	})

	Describe("A resource that was updated", func() {
		BeforeEach(func() {
			Deploy("deployments/single-vm.yml")
		})

		It("has its resource cache, resource cache uses and resource cache volumes cleared out", func() {
			By("setting pipeline that creates resource cache")
			fly("set-pipeline", "-n", "-c", "pipelines/get-task.yml", "-p", "volume-gc-test")

			By("unpausing the pipeline")
			fly("unpause-pipeline", "-p", "volume-gc-test")

			By("triggering the job")
			fly("trigger-job", "-w", "-j", "volume-gc-test/simple-job")

			By("getting the resource cache volumes")
			volumes := flyTable("volumes")
			originalResourceVolumeHandles := []string{}
			for _, volume := range volumes {
				if volume["type"] == "resource" && strings.HasPrefix(volume["identifier"], "time:") {
					originalResourceVolumeHandles = append(originalResourceVolumeHandles, volume["handle"])
				}
			}
			Expect(originalResourceVolumeHandles).To(HaveLen(1))

			By("getting the resource caches")
			var originalResourceCacheID int
			err := psql.Select("id").From("resource_caches").Where("version LIKE ?", fmt.Sprint("%", "time", "%")).RunWith(dbConn).QueryRow().Scan(&originalResourceCacheID)
			Expect(err).ToNot(HaveOccurred())
			Expect(originalResourceCacheID).NotTo(BeZero())

			By("updating pipeline and removing resource")
			fly("set-pipeline", "-n", "-c", "pipelines/get-task-changing-resource.yml", "-p", "volume-gc-test")

			By("eventually expiring the resource cache volumes")
			Eventually(func() []string {
				return volumesByResourceType("time")
			}, 5*time.Minute, 10*time.Second).ShouldNot(ContainElement(originalResourceVolumeHandles[0]))

			By("expiring the resource caches")
			var resourceCacheID int
			err = psql.Select("id").From("resource_caches").Where("version LIKE ?", fmt.Sprint("%", "time", "%")).RunWith(dbConn).QueryRow().Scan(&resourceCacheID)
			// depending on timing either the cache is gone or the new was created for new resource config
			if err != nil {
				Expect(err).To(Equal(sql.ErrNoRows))
			} else {
				Expect(err).ToNot(HaveOccurred())
				Expect(resourceCacheID).NotTo(Equal(originalResourceCacheID))
			}
		})
	})

	Describe("A resource in paused pipeline", func() {
		BeforeEach(func() {
			Deploy("deployments/single-vm.yml")
		})

		It("has its resource cache, resource cache uses and resource cache volumes cleared out", func() {
			By("setting pipeline that creates resource cache")
			fly("set-pipeline", "-n", "-c", "pipelines/get-task-changing-resource.yml", "-p", "volume-gc-test")

			By("unpausing the pipeline")
			fly("unpause-pipeline", "-p", "volume-gc-test")

			By("triggering the job")
			fly("trigger-job", "-w", "-j", "volume-gc-test/simple-job")

			By("getting the resource cache volumes")
			volumes := flyTable("volumes")
			resourceVolumeHandles := []string{}
			for _, volume := range volumes {
				if volume["type"] == "resource" && strings.HasPrefix(volume["identifier"], "time:") {
					resourceVolumeHandles = append(resourceVolumeHandles, volume["handle"])
				}
			}
			Expect(resourceVolumeHandles).To(HaveLen(1))

			By("getting the resource caches")
			var resourceCachesNum int
			err := psql.Select("COUNT(id)").From("resource_caches").Where("version LIKE ?", fmt.Sprint("%", "time", "%")).RunWith(dbConn).QueryRow().Scan(&resourceCachesNum)
			Expect(err).ToNot(HaveOccurred())
			Expect(resourceCachesNum).To(Equal(1))

			By("getting the resource caches uses")
			var resourceCacheUsesNum int
			err = psql.Select("COUNT(*)").From("resource_cache_uses").RunWith(dbConn).QueryRow().Scan(&resourceCacheUsesNum)
			Expect(err).ToNot(HaveOccurred())
			// one for image resource, one for time
			Expect(resourceCacheUsesNum).To(Equal(2))

			By("pausing the pipeline")
			fly("pause-pipeline", "-p", "volume-gc-test")

			By("eventually expiring the resource cache volumes")
			Eventually(func() int {
				return len(volumesByResourceType("time"))
			}, 5*time.Minute, 10*time.Second).Should(BeZero())

			By("expiring the resource caches")
			err = psql.Select("COUNT(id)").From("resource_caches").Where("version LIKE ?", fmt.Sprint("%", "time", "%")).RunWith(dbConn).QueryRow().Scan(&resourceCachesNum)
			Expect(err).ToNot(HaveOccurred())
			Expect(resourceCachesNum).To(BeZero())

			By("expiring the resource caches uses")
			err = psql.Select("COUNT(*)").From("resource_cache_uses").RunWith(dbConn).QueryRow().Scan(&resourceCacheUsesNum)
			Expect(err).ToNot(HaveOccurred())
			Expect(resourceCacheUsesNum).To(BeZero())
		})
	})

	Describe("a resource that has new versions", func() {
		var (
			gitRepoURI string
			gitRepo    GitRepo
		)

		BeforeEach(func() {
			Deploy("deployments/single-vm.yml", "-o", "operations/add-git-server.yml")

			gitRepoURI = fmt.Sprintf("git://%s/some-repo", JobInstance("git_server").IP)
			gitRepo = NewGitRepo(gitRepoURI)
		})

		AfterEach(func() {
			gitRepo.Cleanup()
		})

		It("has its old resource cache, old resource cache uses and old resource cache volumes cleared out", func() {
			By("creating an initial resource version")
			gitRepo.CommitAndPush()

			By("setting pipeline that creates resource cache")
			fly("set-pipeline", "-n", "-c", "pipelines/get-git-resource.yml", "-p", "volume-gc-test", "-v", "some-repo-uri="+gitRepoURI)

			By("unpausing the pipeline")
			fly("unpause-pipeline", "-p", "volume-gc-test")

			By("triggering the job")
			fly("trigger-job", "-w", "-j", "volume-gc-test/simple-job")

			By("getting the resource cache volumes")
			volumes := flyTable("volumes")
			originalResourceVolumeHandles := []string{}
			for _, volume := range volumes {
				if volume["type"] == "resource" && strings.HasPrefix(volume["identifier"], "ref:") {
					originalResourceVolumeHandles = append(originalResourceVolumeHandles, volume["handle"])
				}
			}
			Expect(originalResourceVolumeHandles).To(HaveLen(1))

			By("getting the resource caches")
			var originalResourceCacheVersion string
			err := psql.Select("version").From("resource_caches").Where("version LIKE ?", fmt.Sprint("%", "ref", "%")).RunWith(dbConn).QueryRow().Scan(&originalResourceCacheVersion)
			Expect(err).ToNot(HaveOccurred())

			By("creating a new resource version")
			gitRepo.CommitAndPush()

			By("triggering the job")
			fly("trigger-job", "-w", "-j", "volume-gc-test/simple-job")

			By("eventually expiring the resource cache volume")
			Eventually(func() []string {
				volumes := flyTable("volumes")
				resourceVolumeHandles := []string{}
				for _, volume := range volumes {
					if volume["type"] == "resource" && strings.HasPrefix(volume["identifier"], "ref:") {
						resourceVolumeHandles = append(resourceVolumeHandles, volume["handle"])
					}
				}
				return resourceVolumeHandles
			}, 10*time.Minute, 10*time.Second).ShouldNot(ContainElement(originalResourceVolumeHandles[0]))

			By("expiring the resource caches")
			var newResourceCacheVersion string
			err = psql.Select("version").From("resource_caches").Where("version LIKE ?", fmt.Sprint("%", "ref", "%")).RunWith(dbConn).QueryRow().Scan(&newResourceCacheVersion)
			Expect(err).ToNot(HaveOccurred())
			Expect(newResourceCacheVersion).NotTo(Equal(originalResourceCacheVersion))
		})
	})
})
