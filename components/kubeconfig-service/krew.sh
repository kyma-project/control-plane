#!/bin/bash
KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl


K_CHECKSUM=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -LO "https://dl.k8s.io/$K_CHECKSUM/bin/linux/amd64/kubectl.sha256"

echo "Checked kubectl binary with sha256sum " >> ~/info.txt
echo "$(<kubectl.sha256)  kubectl" | sha256sum -c >> ~/info.txt
rm -rf kubectl.sha256

sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
echo -e "\nkubectl version $KUBECTL_VERSION installed " >> ~/info.txt

echo "First install krew"

set -x; 
cd "$(mktemp -d)" &&
OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')" &&
KREW="krew-${OS}_${ARCH}" &&
curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" &&
tar zxvf "${KREW}.tar.gz" &&
./"${KREW}" install krew

cp -r ${HOME}/.krew/bin/kubectl-krew /usr/local/bin/
export PATH="${PATH}:${HOME}/.krew/bin"
echo "Using krew install oidc-login pulgin"
kubectl krew install oidc-login
kubectl krew list
kubectl oidc-login
