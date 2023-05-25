package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
)

type Download struct {
	url       string
	RemoteStr []string // 远程文件内容
}

// RemoteImageTags 远程仓库中的镜像Tag
type RemoteImageTags struct {
	Repository string   `json:"Repository"`
	Tags       []string `json:"Tags"`
}

// DestImageTags 远程仓库中的镜像Tag
type DestImageTags struct {
	Repository string   `json:"Repository"`
	Tags       []string `json:"Tags"`
}

func NewDockerSync(url string) *Download {
	return &Download{
		url: url,
	}
}

// GetRemoteCtx 获取远程地址文件中存放的需要同步的镜像名称内容
func (d *Download) GetRemoteCtx() {
	// 跳过证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	response, err := client.Get(d.url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("io.ReadAll --- %s", err)
		return
	}

	// 获取远程文件内容进行换行处理
	data := strings.NewReader(string(responseData))
	readerLine := bufio.NewReader(data)

	for {
		readLineStr, err := readerLine.ReadString('\n')

		if len(readLineStr) > 2 {
			// 去除文本中的换行符
			newText := strings.Trim(readLineStr, "\n")
			d.RemoteStr = append(d.RemoteStr, newText)
		}

		if err == io.EOF {
			break
		}
	}
}

func (d *Download) CopyImage(destHub string) {
	for _, v := range d.RemoteStr {
		// 获取远程镜像的标签
		remoteTags, err := getRemoteImageTags(v)
		if err != nil {
			fmt.Printf("Failed to get remote image tags: %s\n", err)
			continue
		}

		next := filepath.Base(remoteTags.Repository)

		// 检查本地仓库是否有当前镜像
		destImageTags, err := getDestImageTags(destHub, next)
		if err != nil {
			fmt.Printf("Failed to get destination image tags: %s\n", err)
			continue
		}

		// 对比标签数组差异
		diffTags := difference(remoteTags.Tags, destImageTags.Tags)

		destName := getDestName(remoteTags.Repository)

		fmt.Printf("远程仓库和本地仓库差异的标签：%s\n", diffTags)

		// 复制镜像
		for _, tag := range diffTags {
			err = copyImage(remoteTags.Repository, destHub, destName, tag)
			if err != nil {
				fmt.Printf("Failed to copy image: %s\n", err)
				continue
			}

			fmt.Printf("远程仓库镜像 --> docker://%s:%s\n", remoteTags.Repository, tag)
			fmt.Printf("本地仓库镜像 --> docker://%s%s:%s\n", destHub, destName, tag)
		}
	}
}

// getRemoteImageTags 获取远程镜像的标签
func getRemoteImageTags(imageURL string) (*RemoteImageTags, error) {
	remoteCmd := exec.Command("skopeo", "list-tags", "docker://"+imageURL, "--tls-verify=false")

	var remoteStdout, remoteStderr bytes.Buffer
	remoteCmd.Stdout = &remoteStdout // 标准输出
	remoteCmd.Stderr = &remoteStderr // 标准错误

	err := remoteCmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute remoteCmd: %s", err)
	}

	var remoteTags RemoteImageTags
	if remoteStdout.String() != "" {
		err = json.Unmarshal(remoteStdout.Bytes(), &remoteTags)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal remoteTags: %s", err)
		}
	} else {
		fmt.Println(remoteStderr.String())
		return nil, fmt.Errorf("remoteCmd failed: %s", err)
	}

	return &remoteTags, nil
}

// getDestImageTags 检查本地仓库是否有当前镜像
func getDestImageTags(destHub, imageName string) (*DestImageTags, error) {
	destCmd := exec.Command("skopeo", "list-tags", "docker://"+destHub+imageName, "--tls-verify=false")

	var destStdout, destStderr bytes.Buffer
	destCmd.Stdout = &destStdout // 标准输出
	destCmd.Stderr = &destStderr // 标准错误

	err := destCmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to execute destCmd: %s", err)
	}

	var destImageTags DestImageTags
	if destStdout.String() != "" {
		err = json.Unmarshal(destStdout.Bytes(), &destImageTags)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal destImageTags: %s", err)
		}
	}

	return &destImageTags, nil
}

// copyImage 复制镜像
func copyImage(srcRepo, destHub, destName, tag string) error {
	pushCmd := exec.Command("skopeo",
		"copy",
		"--insecure-policy",
		"--src-tls-verify=false",
		"--dest-tls-verify=false",
		"-q",
		"docker://"+srcRepo+":"+tag,
		"docker://"+destHub+destName+":"+tag)

	err := pushCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute pushCmd: %s", err)
	}

	return nil
}

// getDestName 根据远程仓库URL获取目标名称
func getDestName(repoURL string) string {
	count := strings.Count(repoURL, "/")
	if count == 1 {
		return filepath.Base(repoURL)
	}

	destName := filepath.Base(repoURL)
	if count == 2 {
		if strings.Contains(repoURL, "registry.k8s.io") && !strings.Contains(repoURL, "coredns") {
			begin := strings.ReplaceAll(repoURL, "registry.k8s.io/", "")
			last := strings.ReplaceAll(begin, "-", "_")
			destName = strings.ReplaceAll(last, "/", "_")
		} else if strings.Contains(repoURL, "quay.io") {
			begin := strings.ReplaceAll(repoURL, "quay.io/", "")
			last := strings.ReplaceAll(begin, "-", "_")
			destName = strings.ReplaceAll(last, "/", "_")
		} else if strings.Contains(repoURL, "gcr.io") {
			begin := strings.ReplaceAll(repoURL, "gcr.io/", "")
			last := strings.ReplaceAll(begin, "-", "_")
			destName = strings.ReplaceAll(last, "/", "_")
		} else if strings.Contains(repoURL, "docker.io") {
			begin := strings.ReplaceAll(repoURL, "docker.io/", "")
			last := strings.ReplaceAll(begin, "-", "_")
			destName = strings.ReplaceAll(last, "/", "_")
		}
	}

	return destName
}

// 求交集
func intersect(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	for _, v := range slice1 {
		m[v]++
	}

	for _, v := range slice2 {
		times, _ := m[v]
		if times == 1 {
			nn = append(nn, v)
		}
	}
	return nn
}

// 求差集 slice1-并集
func difference(slice1, slice2 []string) []string {
	m := make(map[string]int)
	nn := make([]string, 0)
	inter := intersect(slice1, slice2)
	for _, v := range inter {
		m[v]++
	}

	for _, value := range slice1 {
		times, _ := m[value]
		if times == 0 {
			nn = append(nn, value)
		}
	}
	return nn
}
