package enterprise

import (
	"os"
	"path/filepath"
)

const githubEnterpriseWorkflow = `name: ZED Enterprise Gate

on:
  pull_request:
  push:
    branches: [ main, master ]

jobs:
  zed-enterprise:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Build ZED
        run: go build -o zed ./cmd/zed
      - name: Test
        run: go test ./...
      - name: Enterprise SARIF
        run: ./zed --enterprise-sarif zed-enterprise.sarif || true
      - name: Enterprise Policy Gate
        run: ./zed --enterprise-policy-gate
      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: zed-enterprise.sarif
`

const gitlabEnterpriseCI = `zed_enterprise_gate:
  image: golang:1.23
  stage: test
  script:
    - go build -o zed ./cmd/zed
    - go test ./...
    - ./zed --enterprise-sarif zed-enterprise.sarif || true
    - ./zed --enterprise-policy-gate
  artifacts:
    when: always
    paths:
      - zed-enterprise.sarif
`

func WriteCITemplates(root string) ([]string, error) {
	files := map[string]string{
		filepath.Join(root, ".github", "workflows", "zed-enterprise.yml"): githubEnterpriseWorkflow,
		filepath.Join(root, ".gitlab-ci-zed-enterprise.yml"): gitlabEnterpriseCI,
	}
	var written []string
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { return written, err }
		if err := os.WriteFile(path, []byte(content), 0644); err != nil { return written, err }
		written = append(written, path)
	}
	return written, nil
}
