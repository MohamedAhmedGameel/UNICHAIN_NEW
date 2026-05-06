# Windows + WSL2 Setup

How to get the project from Windows into WSL2 and install all prerequisites.  
Do this **once** on a new machine.

---

## 1. Enable WSL2 on Windows 11

Open **PowerShell as Administrator** and run:

```powershell
wsl --install
wsl --set-default-version 2
```

Restart your machine. Then install Ubuntu from the Microsoft Store if it wasn't installed automatically.

Open Ubuntu from Start Menu and create your Linux username and password.

---

## 2. Install Docker Desktop

Download from: https://www.docker.com/products/docker-desktop

During install:
- ✅ Enable WSL2 backend
- ✅ Enable integration with your Ubuntu distro

After install, open Docker Desktop → Settings → Resources → WSL Integration → toggle on your Ubuntu distro.

Verify inside WSL2:

```bash
docker ps
# Must return a table header, not an error
```

---

## 3. Copy Project from Windows into WSL2

**Always work inside the WSL2 filesystem (`~/`), never from `/mnt/c/`.** Working from `/mnt/c/` causes file permission errors, CRLF line ending corruption, and slow I/O.

### Option A — Copy with `cp`

Open WSL2 terminal and run:

```bash
# Replace YourWindowsUsername with your actual Windows username
cp -r /mnt/c/Users/YourWindowsUsername/path/to/hyperledger ~/hyperledger

# Verify the copy
ls ~/hyperledger/
# Expected: chaincode/  network/
```

### Option B — Copy with `rsync` (faster for large projects)

```bash
rsync -av --progress \
  /mnt/c/Users/YourWindowsUsername/path/to/hyperledger/ \
  ~/hyperledger/
```

### Fix Line Endings After Copy

Windows saves files with `\r\n` (CRLF). Linux needs `\n` (LF). Fix all Go and config files:

```bash
find ~/hyperledger/chaincode -name "*.go" -exec sed -i 's/\r//' {} +
find ~/hyperledger/network -name "*.sh" -exec sed -i 's/\r//' {} +
find ~/hyperledger/network -name "*.yaml" -exec sed -i 's/\r//' {} +
find ~/hyperledger/network -name "*.json" -exec sed -i 's/\r//' {} +
```

### Fix Permissions on Shell Scripts

```bash
chmod +x ~/hyperledger/network/*.sh
chmod +x ~/hyperledger/network/scripts/*.sh
```

---

## 4. Install Go Inside WSL2

**Do not use the Windows Go installation.** The chaincode must be compiled by the Linux Go toolchain.

```bash
# Download Go 1.21 for Linux
wget https://go.dev/dl/go1.21.13.linux-amd64.tar.gz

# Install
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# Verify — must show Linux path, NOT /mnt/c/
which go
# /usr/local/go/bin/go  ✓

go version
# go version go1.21.x linux/amd64  ✓
```

---

## 5. Install Python3

Required for the package ID extraction step in deploy scripts.

```bash
sudo apt update
sudo apt install -y python3
python3 --version
```

---

## 6. Add Fabric Binaries to PATH Permanently

```bash
echo 'export PATH=$PATH:$HOME/hyperledger/network/fabric-samples/bin' >> ~/.bashrc
echo 'export FABRIC_CFG_PATH=$HOME/hyperledger/network/fabric-samples/config' >> ~/.bashrc
source ~/.bashrc

# Verify
which peer
# /home/youruser/hyperledger/network/fabric-samples/bin/peer  ✓
```

---

## 7. Add All Environment Variables Permanently

```bash
cat >> ~/.bashrc << 'EOF'

# UniChain environment
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_TLS_ROOTCERT_FILE=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/peers/peer0.org1.university.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=$HOME/hyperledger/network/organizations/peerOrganizations/org1.university.com/users/Admin@org1.university.com/msp
export ORDERER_CA=$HOME/hyperledger/network/organizations/ordererOrganizations/university.com/orderers/orderer.university.com/msp/tlscacerts/tlsca.university.com-cert.pem
EOF

source ~/.bashrc
```

---

## 8. Verify Everything

```bash
echo "Go:     $(go version)"
echo "Peer:   $(peer version 2>&1 | head -1)"
echo "Docker: $(docker --version)"
echo "Python: $(python3 --version)"
docker ps   # must not error
```

All four should return version strings with no errors. If Docker errors, make sure Docker Desktop is running on Windows.

---

## Updating the Project Later

When you edit code on Windows and want to sync changes to WSL2:

```bash
# Re-copy changed files
cp /mnt/c/Users/YourWindowsUsername/path/to/hyperledger/chaincode/university/chaincode.go \
   ~/hyperledger/chaincode/university/chaincode.go

# Fix line endings on the changed file
sed -i 's/\r//' ~/hyperledger/chaincode/university/chaincode.go

# Verify it still compiles
cd ~/hyperledger/chaincode/university
go build ./...
```

Or use VS Code with the **Remote - WSL** extension to edit files directly inside WSL2, which avoids the copy step entirely.
