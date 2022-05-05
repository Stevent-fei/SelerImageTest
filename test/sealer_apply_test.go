package test

import (
	"blog/test/testhelper"
	"fmt"
	. "github.com/onsi/ginkgo"
	"strings"
	"time"

	"blog/test/suites/apply"
	"blog/test/testhelper/settings"
)

var _ = Describe("sealer apply", func() {
	Context("start apply calico", func() {
		rawClusterFilePath := apply.GetRawClusterFilePath()
		rawCluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
		rawCluster.Spec.Image = settings.TestImageName
		rawCluster.Spec.Env = settings.CalicoEnv
		BeforeEach(func() {
			if rawCluster.Spec.Image != settings.TestImageName {
				rawCluster.Spec.Image = settings.TestImageName
				apply.MarshalClusterToFile(rawClusterFilePath, rawCluster)
			}
		})

		Context("check regular scenario that provider is bare metal, executes machine is master0", func() {
			var tempFile string
			BeforeEach(func() {
				tempFile = testhelper.CreateTempFile()
			})

			AfterEach(func() {
				testhelper.RemoveTempFile(tempFile)
			})
			It("init, clean up", func() {
				By("start to prepare infra")
				cluster := rawCluster.DeepCopy()
				cluster.Spec.Provider = settings.AliCloud
				cluster.Spec.Image = settings.TestImageName
				cluster = apply.CreateAliCloudInfraAndSave(cluster, tempFile)
				defer apply.CleanUpAliCloudInfra(cluster)
				sshClient := testhelper.NewSSHClientByCluster(cluster)
				testhelper.CheckFuncBeTrue(func() bool {
					err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
					return err == nil
				}, settings.MaxWaiteTime)

				By("start to init cluster")
				apply.GenerateClusterfile(tempFile)
				apply.SendAndApplyCluster(sshClient, tempFile)

				By("Wait for the cluster to be ready", func() {
					apply.WaitAllNodeRunningBySSH(sshClient.SSH,sshClient.RemoteHostIP)
				})
				By("start to delete cluster")
				//err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, apply.SealerDeleteCmd(tempFile))
				//testhelper.CheckErr(err)
				apply.SealerDelete()
				By("apply.SealerDelete()")
				time.Sleep(10 *time.Second)
				By("sealer run calico", func() {
					masters := strings.Join(cluster.Spec.Masters.IPList, ",")
					nodes := strings.Join(cluster.Spec.Nodes.IPList, ",")
					apply.SendAndRunCluster(sshClient, tempFile, masters, nodes, cluster.Spec.SSH.Passwd)
					apply.CheckNodeNumWithSSH(sshClient, 2)
				})
				fmt.Println("test finish")
			})
		})
	})

	//Context("start apply hybridnet", func() {
	//	rawClusterFilePath := apply.GetRawClusterFilePath()
	//	rawCluster := apply.LoadClusterFileFromDisk(rawClusterFilePath)
	//	rawCluster.Spec.Image = settings.TestImageName
	//	rawCluster.Spec.Env = settings.HybridnetEnv
	//	BeforeEach(func() {
	//		if rawCluster.Spec.Image != settings.TestImageName {
	//			rawCluster.Spec.Image = settings.TestImageName
	//			apply.MarshalClusterToFile(rawClusterFilePath, rawCluster)
	//		}
	//	})
	//
	//	Context("check regular scenario that provider is bare metal, executes machine is master0", func() {
	//		var tempFile string
	//		BeforeEach(func() {
	//			tempFile = testhelper.CreateTempFile()
	//		})
	//
	//		AfterEach(func() {
	//			testhelper.RemoveTempFile(tempFile)
	//		})
	//		It("init, clean up", func() {
	//			By("start to prepare infra")
	//			cluster := rawCluster.DeepCopy()
	//			cluster.Spec.Provider = settings.AliCloud
	//			cluster.Spec.Image = settings.TestImageName
	//			cluster = apply.CreateAliCloudInfraAndSave(cluster, tempFile)
	//			defer apply.CleanUpAliCloudInfra(cluster)
	//			sshClient := testhelper.NewSSHClientByCluster(cluster)
	//			testhelper.CheckFuncBeTrue(func() bool {
	//				err := sshClient.SSH.Copy(sshClient.RemoteHostIP, settings.DefaultSealerBin, settings.DefaultSealerBin)
	//				return err == nil
	//			}, settings.MaxWaiteTime)
	//
	//			By("start to init cluster")
	//			apply.GenerateClusterfile(tempFile)
	//			apply.SendAndApplyCluster(sshClient, tempFile)
	//			apply.CheckNodeNumWithSSH(sshClient, 2)
	//
	//			By("Wait for the cluster to be ready", func() {
	//				apply.WaitAllNodeRunningBySSH(sshClient.SSH,sshClient.RemoteHostIP)
	//			})
	//			By("start to delete cluster")
	//			err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, apply.SealerDeleteCmd(tempFile))
	//			testhelper.CheckErr(err)
	//		})
	//	})
	//})
})
