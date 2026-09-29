package smoke_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testNodejs(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose().WithNoColor()
		docker = occam.NewDocker()
	})

	context("detects a Nodejs app", func() {
		var (
			imageIDs  []string
			container occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			for _, id := range imageIDs {
				Expect(docker.Image.Remove.Execute(id)).To(Succeed())
			}
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds successfully", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "nodejs", "npm"))
			Expect(err).NotTo(HaveOccurred())

			build := pack.Build.
				WithNetwork("host").
				WithPullPolicy("always").
				WithBuilder(Builder)

			_, firstLogs, err := build.Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), firstLogs.String)

			secondImage, secondLogs, err := build.Execute(name, source)
			Expect(err).ToNot(HaveOccurred(), secondLogs.String)
			imageIDs = append(imageIDs, secondImage.ID)

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(secondImage.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			Expect(secondLogs).To(ContainLines(ContainSubstring("Paketo Buildpack for Node Engine")))
			Expect(secondLogs).To(ContainLines(ContainSubstring("Paketo Buildpack for NPM Install")))
			Expect(secondLogs).To(ContainLines(ContainSubstring("Paketo Buildpack for NPM Start")))
			Expect(secondLogs).To(ContainLines(ContainSubstring("[extender (build)] Enabling module streams")))
			Expect(secondLogs).To(ContainLines(MatchRegexp(`nodejs:\d+`)))
			Expect(secondLogs).To(ContainLines(ContainSubstring("[extender (build)]   Node no longer requested by plan")))
		})
	})
}
