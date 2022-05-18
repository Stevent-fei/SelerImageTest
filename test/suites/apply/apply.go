package apply

import (
	"blog/test/testhelper"
	"blog/test/testhelper/settings"
	"fmt"
	"github.com/alibaba/sealer/pkg/infra"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
	"github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func LoadClusterFileFromDisk(clusterFilePath string) *v1.Cluster {
	clusters, err := utils.DecodeCluster(clusterFilePath)
	testhelper.CheckErr(err)
	testhelper.CheckNotNil(clusters[0])
	return &clusters[0]
}

func getFixtures() string {
	pwd := settings.DefaultTestEnvDir
	return filepath.Join(pwd, "suites", "apply", "fixtures")
}

func GetRawClusterFilePath() string {
	fixtures := getFixtures()
	return filepath.Join(fixtures, "cluster_file_for_test.yaml")
}

func CreateAliCloudInfraAndSave(cluster *v1.Cluster, clusterFile string) *v1.Cluster {
	CreateAliCloudInfra(cluster)
	//save used cluster file
	cluster.Spec.Provider = settings.BAREMETAL
	MarshalClusterToFile(clusterFile, cluster)
	cluster.Spec.Provider = settings.AliCloud
	return cluster
}

func CreateAliCloudInfra(cluster *v1.Cluster) {
	cluster.DeletionTimestamp = nil
	infraManager, err := infra.NewDefaultProvider(cluster)
	testhelper.CheckErr(err)
	err = infraManager.Apply()
	testhelper.CheckErr(err)
}

func MarshalClusterToFile(ClusterFile string, cluster *v1.Cluster) {
	err := testhelper.MarshalYamlToFile(ClusterFile, &cluster)
	testhelper.CheckErr(err)
	testhelper.CheckNotNil(cluster)
}

func CleanUpAliCloudInfra(cluster *v1.Cluster) {
	if cluster == nil {
		return
	}
	if cluster.Spec.Provider != settings.AliCloud {
		cluster.Spec.Provider = settings.AliCloud
	}
	t := metav1.Now()
	cluster.DeletionTimestamp = &t
	infraManager, err := infra.NewDefaultProvider(cluster)
	testhelper.CheckErr(err)
	err = infraManager.Apply()
	testhelper.CheckErr(err)
}

func SendAndRunCluster(sshClient *testhelper.SSHClient, clusterFile string, joinMasters, joinNodes, passwd string) {
	SendAndRemoteExecCluster(sshClient, clusterFile, SealerRunCalicoCmd(joinMasters, joinNodes, passwd, ""))
}

func SendAndRunHybirdnetCluster(sshClient *testhelper.SSHClient, clusterFile string, joinMasters, joinNodes, passwd string) {
	SendAndRemoteExecCluster(sshClient, clusterFile, SealerRunHybridnetCmd(joinMasters, joinNodes, passwd, ""))
}

func SendAndRemoteExecCluster(sshClient *testhelper.SSHClient, clusterFile string, remoteCmd string) {
	// send tmp cluster file to remote server and run apply cmd
	gomega.Eventually(func() bool {
		err := sshClient.SSH.Copy(sshClient.RemoteHostIP, clusterFile, clusterFile)
		return err == nil
	}, settings.MaxWaiteTime).Should(gomega.BeTrue())
	err := sshClient.SSH.CmdAsync(sshClient.RemoteHostIP, remoteCmd)
	testhelper.CheckErr(err)
}

func SealerRunCalicoCmd(masters, nodes, passwd string, provider string) string {
	if masters != "" {
		masters = fmt.Sprintf("-m %s", masters)
	}
	if nodes != "" {
		nodes = fmt.Sprintf("-n %s", nodes)
	}
	if passwd != "" {
		passwd = fmt.Sprintf("-p %s", passwd)
	}
	if provider != "" {
		provider = fmt.Sprintf("--provider %s", provider)
	}
	return fmt.Sprintf("%s run %s -e %s %s %s %s %s -d", settings.DefaultSealerBin, settings.TestImageName, settings.CustomCalicoEnv , masters, nodes, passwd, provider)
}

func SealerRunHybridnetCmd(masters, nodes, passwd string, provider string) string {
	if masters != "" {
		masters = fmt.Sprintf("-m %s", masters)
	}
	if nodes != "" {
		nodes = fmt.Sprintf("-n %s", nodes)
	}
	if passwd != "" {
		passwd = fmt.Sprintf("-p %s", passwd)
	}
	if provider != "" {
		provider = fmt.Sprintf("--provider %s", provider)
	}
	return fmt.Sprintf("%s run %s -e %s %s %s %s %s -d", settings.DefaultSealerBin, settings.TestImageName, settings.CustomhybridnetEnv , masters, nodes, passwd, provider)
}

func NodeRunCmd() string {
	return fmt.Sprintf("sudo bash %s",settings.LoadPath)
}

func Permissions() string {
	return fmt.Sprintf("cp .kube/config /tmp/kubeconfig && chmod +x /tmp/kubeconfig")
}

func GetE2eTestFile() string {
	return fmt.Sprintf("wget https://sealer.oss-cn-beijing.aliyuncs.com/e2e/e2e.tar && tar -xvf e2e.tar")
}

func ExecE2eTestFile() string {
	return fmt.Sprintf("sudo bash run.sh && sudo bash get-log.sh")
}

// CheckNodeNumWithSSH check node mum of remote cluster;for bare metal apply
func CheckNodeNumWithSSH(sshClient *testhelper.SSHClient, expectNum int) {
	if sshClient == nil {
		return
	}
	cmd := "kubectl get nodes | wc -l"
	result, err := sshClient.SSH.CmdToString(sshClient.RemoteHostIP, cmd, "")
	testhelper.CheckErr(err)
	num, err := strconv.Atoi(strings.ReplaceAll(result, "\n", ""))
	testhelper.CheckErr(err)
	testhelper.CheckEqual(num, expectNum+1)
}

func GenerateClusterfile(clusterfile string) {
	cluster := LoadClusterFileFromDisk(clusterfile)
	data, err := yaml.Marshal(cluster)
	testhelper.CheckErr(err)
	testhelper.CheckNotNil(data)
}

func SendAndApplyCluster(sshClient *testhelper.SSHClient, clusterFile string) {
	SendAndRemoteExecCluster(sshClient, clusterFile, SealerApplyCmd(clusterFile))
}

func SendAndLoad(sshClient *testhelper.SSHClient, clusterFile string) {
	SendAndRemoteExecCluster(sshClient, clusterFile, SealerApplyCmd(clusterFile))
}

func SealerApplyCmd(clusterFile string) string {
	return fmt.Sprintf("%s apply -f %s --force -d", settings.DefaultSealerBin, clusterFile)
}

func WaitAllNodeRunningBySSH(s ssh.Interface, masterIp string) {
	time.Sleep(30 * time.Second)
	err := utils.Retry(10,5 * time.Second, func() error {
		result, err := s.CmdToString(masterIp, "kubectl get node", "")
		if err != nil{
			return err
		}
		if strings.Contains(result, "NotReady") {
			return fmt.Errorf("node not ready: \n %s", result)
		}
		return nil
	})
	testhelper.CheckErr(err)
}

func SealerDeleteCmd(clusterFile string) string {
	return fmt.Sprintf("%s delete -f %s --force -d", settings.DefaultSealerBin,clusterFile)
}

func SealerDelete() string {
	return fmt.Sprintf("%s delete -a --force -d", settings.DefaultSealerBin)
}

func GetE2eTest() string {
	return fmt.Sprintf("wget https://sealer.oss-cn-beijing.aliyuncs.com/kubernetes_e2e_images_v1.20.0.tar.gz")
}

func GetShell() string {
	path := settings.LoadPath
	return filepath.Join(path, "suites", "apply", "fixtures")
}

func GetLoadFilePath() string {
	fixtures := GetShell()
	return filepath.Join(fixtures, "load.sh")
}

