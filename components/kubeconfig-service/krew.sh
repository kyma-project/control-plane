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
curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/krew.tar.gz" &&
tar zxvf krew.tar.gz &&
KREW=./krew-"${OS}_${ARCH}" &&
"$KREW" install krew
export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"

echo "Using krew install oidc-login pulgin"
kubectl krew install oidc-login