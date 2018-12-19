package api_test

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/concourse/baggageclaim"
	"github.com/concourse/concourse/atc/api/accessor/accessorfakes"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/db/dbfakes"
	"github.com/concourse/concourse/atc/worker/workerfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artifacts API", func() {
	var fakeaccess *accessorfakes.FakeAccess

	BeforeEach(func() {
		fakeaccess = new(accessorfakes.FakeAccess)
	})

	JustBeforeEach(func() {
		fakeAccessor.CreateReturns(fakeaccess)
	})

	Describe("POST /api/v1/teams/:team_name/artifacts", func() {
		var request *http.Request
		var response *http.Response

		BeforeEach(func() {
			fakeaccess = new(accessorfakes.FakeAccess)
			fakeaccess.IsAuthenticatedReturns(true)

			fakeAccessor.CreateReturns(fakeaccess)
		})

		JustBeforeEach(func() {
			var err error
			request, err = http.NewRequest("POST", server.URL+"/api/v1/teams/some-team/artifacts", bytes.NewBuffer([]byte("some-data")))
			Expect(err).NotTo(HaveOccurred())

			request.Header.Set("Content-Type", "application/json")

			response, err = client.Do(request)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when not authenticated", func() {
			BeforeEach(func() {
				fakeaccess.IsAuthenticatedReturns(false)
			})

			It("returns 401 Unauthorized", func() {
				Expect(response.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when not authorized", func() {
			BeforeEach(func() {
				fakeaccess.IsAuthorizedReturns(false)
			})

			It("returns 403 Forbidden", func() {
				Expect(response.StatusCode).To(Equal(http.StatusForbidden))
			})
		})

		Context("when authorized", func() {
			BeforeEach(func() {
				fakeaccess.IsAuthorizedReturns(true)
			})

			Context("when creating a volume fails", func() {
				BeforeEach(func() {
					fakeWorkerClient.CreateVolumeReturns(nil, errors.New("nope"))
				})

				It("returns 500 InternalServerError", func() {
					Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
				})
			})

			Context("when creating a volume succeeds", func() {
				var fakeVolume *workerfakes.FakeVolume

				BeforeEach(func() {
					fakeVolume = new(workerfakes.FakeVolume)
					fakeVolume.InitializeArtifactReturns(nil, errors.New("nope"))

					fakeWorkerClient.CreateVolumeReturns(fakeVolume, nil)
				})

				It("creates the volume using the worker client", func() {
					Expect(fakeWorkerClient.CreateVolumeCallCount()).To(Equal(1))

					_, volumeSpec, teamID, volumeType := fakeWorkerClient.CreateVolumeArgsForCall(0)
					Expect(volumeSpec.Strategy).To(Equal(baggageclaim.EmptyStrategy{}))
					Expect(teamID).To(Equal(734))
					Expect(volumeType).To(Equal(db.VolumeTypeArtifact))
				})

				Context("when associating a volume with an artifact fails", func() {
					BeforeEach(func() {
						fakeVolume.InitializeArtifactReturns(nil, errors.New("nope"))
					})

					It("returns 500 InternalServerError", func() {
						Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
					})
				})

				Context("when associating a volume with an artifact succeeds", func() {
					var fakeWorkerArtifact *dbfakes.FakeWorkerArtifact

					BeforeEach(func() {
						fakeWorkerArtifact = new(dbfakes.FakeWorkerArtifact)
						fakeWorkerArtifact.IDReturns(0)
						fakeWorkerArtifact.CreatedAtReturns(time.Now())

						fakeVolume.InitializeArtifactReturns(fakeWorkerArtifact, nil)
					})

					It("invokes the initialization of an artifact on a volume", func() {
						Expect(fakeVolume.InitializeArtifactCallCount()).To(Equal(1))

						path, checksum := fakeVolume.InitializeArtifactArgsForCall(0)
						Expect(path).To(Equal("/"))
						Expect(checksum).To(Equal(""))
					})

					Context("when streaming in data to a volume fails", func() {
						BeforeEach(func() {
							fakeVolume.StreamInReturns(errors.New("nope"))
						})

						It("returns 500 InternalServerError", func() {
							Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
						})
					})

					Context("when streaming in data to a volume succeeds", func() {

						BeforeEach(func() {
							fakeVolume.StreamInReturns(nil)

							fakeVolume.StreamInStub = func(path string, body io.Reader) error {
								Expect(path).To(Equal("/"))

								contents, err := ioutil.ReadAll(body)
								Expect(err).ToNot(HaveOccurred())

								Expect(contents).To(Equal([]byte("some-data")))
								return nil
							}
						})

						It("streams in the user contents to the new volume", func() {
							Expect(fakeVolume.StreamInCallCount()).To(Equal(1))
						})

						Context("when the request succeeds", func() {

							It("returns 201 Created", func() {
								Expect(response.StatusCode).To(Equal(http.StatusCreated))
							})

							It("returns Content-Type 'application/json'", func() {
								Expect(response.Header.Get("Content-Type")).To(Equal("application/json"))
							})

							It("returns the artifact record", func() {
								body, err := ioutil.ReadAll(response.Body)
								Expect(err).NotTo(HaveOccurred())

								Expect(body).To(MatchJSON(`{
									"id": 0,
									"path": "/",
									"created_at": 42,
									"checksum": ""
								}`))
							})
						})
					})
				})
			})
		})

	})

	FDescribe("GET /api/v1/teams/:team_name/artifacts/:artifact_id", func() {
		var response *http.Response

		BeforeEach(func() {
			fakeaccess = new(accessorfakes.FakeAccess)
			fakeaccess.IsAuthenticatedReturns(true)

			fakeAccessor.CreateReturns(fakeaccess)
		})

		JustBeforeEach(func() {
			var err error
			response, err = http.Get(server.URL + "/api/v1/teams/some-team/artifacts/18")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when not authenticated", func() {
			BeforeEach(func() {
				fakeaccess.IsAuthenticatedReturns(false)
			})

			It("returns 401 Unauthorized", func() {
				Expect(response.StatusCode).To(Equal(http.StatusUnauthorized))
			})
		})

		Context("when not authorized", func() {
			BeforeEach(func() {
				fakeaccess.IsAuthorizedReturns(false)
			})

			It("returns 403 Forbidden", func() {
				Expect(response.StatusCode).To(Equal(http.StatusForbidden))
			})
		})

		Context("when authorized", func() {
			BeforeEach(func() {
				fakeaccess.IsAuthorizedReturns(true)
			})

			It("uses the artifactID to fetch the db volume record", func() {
				Expect(dbTeam.FindVolumeForWorkerArtifactCallCount()).To(Equal(1))

				artifactID := dbTeam.FindVolumeForWorkerArtifactArgsForCall(0)
				Expect(artifactID).To(Equal(18))
			})

			Context("when retrieving db artifact volume fails", func() {
				BeforeEach(func() {
					dbTeam.FindVolumeForWorkerArtifactReturns(nil, false, errors.New("nope"))
				})

				It("errors", func() {
					Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
				})
			})

			Context("when the db artifact volume is not found", func() {
				BeforeEach(func() {
					dbTeam.FindVolumeForWorkerArtifactReturns(nil, false, nil)
				})

				It("returns 404", func() {
					Expect(response.StatusCode).To(Equal(http.StatusNotFound))
				})
			})

			Context("when the db artifact volume is found", func() {
				var fakeVolume *dbfakes.FakeCreatedVolume

				BeforeEach(func() {
					fakeVolume = new(dbfakes.FakeCreatedVolume)
					fakeVolume.HandleReturns("some-handle")

					dbTeam.FindVolumeForWorkerArtifactReturns(fakeVolume, true, nil)
				})

				It("uses the volume handle to lookup the worker volume", func() {
					Expect(fakeWorkerClient.LookupVolumeCallCount()).To(Equal(1))

					_, handle := fakeWorkerClient.LookupVolumeArgsForCall(0)
					Expect(handle).To(Equal("some-handle"))
				})

				Context("when the worker client errors", func() {
					BeforeEach(func() {
						fakeWorkerClient.LookupVolumeReturns(nil, false, errors.New("nope"))
					})

					It("returns 500", func() {
						Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
					})
				})

				Context("when the worker client can't find the volume", func() {
					BeforeEach(func() {
						fakeWorkerClient.LookupVolumeReturns(nil, false, nil)
					})

					It("returns 404", func() {
						Expect(response.StatusCode).To(Equal(http.StatusNotFound))
					})
				})

				Context("when the worker client finds the volume", func() {
					var fakeWorkerVolume *workerfakes.FakeVolume

					BeforeEach(func() {
						reader := ioutil.NopCloser(bytes.NewReader([]byte("")))

						fakeWorkerVolume = new(workerfakes.FakeVolume)
						fakeWorkerVolume.StreamOutReturns(reader, nil)

						fakeWorkerClient.LookupVolumeReturns(fakeWorkerVolume, true, nil)
					})

					It("streams out the contents of the volume from the root path", func() {
						Expect(fakeWorkerVolume.StreamOutCallCount()).To(Equal(1))

						path := fakeWorkerVolume.StreamOutArgsForCall(0)
						Expect(path).To(Equal("/"))
					})

					Context("when streaming volume contents fails", func() {
						BeforeEach(func() {
							fakeWorkerVolume.StreamOutReturns(nil, errors.New("nope"))
						})

						It("returns 500", func() {
							Expect(response.StatusCode).To(Equal(http.StatusInternalServerError))
						})
					})

					Context("when streaming volume contents succeeds", func() {
						BeforeEach(func() {
							reader := ioutil.NopCloser(bytes.NewReader([]byte("some-content")))
							fakeWorkerVolume.StreamOutReturns(reader, nil)
						})

						It("returns 200", func() {
							Expect(response.StatusCode).To(Equal(http.StatusOK))
						})

						It("returns Content-Type 'application/octet-stream'", func() {
							Expect(response.Header.Get("Content-Type")).To(Equal("application/octet-stream"))
						})

						It("returns the contents of the volume", func() {
							Expect(ioutil.ReadAll(response.Body)).To(Equal([]byte("some-content")))
						})
					})
				})
			})
		})
	})
})
