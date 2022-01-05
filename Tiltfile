allow_k8s_contexts('kind-kind')

load('ext://restart_process', 'docker_build_with_restart')
load('ext://cert_manager', 'deploy_cert_manager')
def capi():
    local("sops -d clusterctl.yaml > tmp.yaml")
    local("./capi.sh")
    local("rm tmp.yaml")

def create_config():
    local("cp config/manager/config.example.yaml config/manager/config.yaml")
    # if os.path.exists('config/manager/config.yaml') == False:
    #     local("sops -d config/manager/config.example.yaml > config/manager/config.yaml")

def kubebuilder(DOMAIN, IMG='controller:latest', CONTROLLERGEN='crd:trivialVersions=true rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases;'):

    DOCKERFILE = '''FROM golang:alpine
    WORKDIR /
    COPY ./bin/manager /
    CMD ["/manager"]
    '''

    def yaml():
        y = decode_yaml_stream(local('cd config/manager; kustomize edit set image controller=' + IMG + '; cd ../..; kustomize build config/default'))
        for m in y:
            if m["kind"] == "Deployment":
                m["spec"]["template"]["spec"]["securityContext"]["runAsUser"] = 0
        return encode_yaml_stream(y)
    
    def manifests():
        return 'bin/controller-gen ' + CONTROLLERGEN
    
    def generate():
        return 'bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./...";'
    
    def vetfmt():
        return 'go vet ./...; go fmt ./...'
    
    def binary():
        return 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o bin/manager main.go'

    installed = local("which kubebuilder")
    print("kubebuilder is present:", installed)

    DIRNAME = os.path.basename(os.getcwd())
    # if kubebuilder
    if os.path.exists('go.mod') == False:
        local("go mod init %s" % DIRNAME)
    
    if os.path.exists('PROJECT') == False:
        local("kubebuilder init --domain %s" % DOMAIN)

    local(manifests() + generate())
    k8s_yaml(yaml())
    
    deps = ['controllers', 'main.go']
    deps.append('api')
    local_resource('Watch&Compile', generate() + binary(), deps=deps, ignore=['*/*/zz_generated.deepcopy.go'])
    
    local_resource('Sample YAML', 'kubectl apply -f ./config/samples/', deps=["./config/samples"], resource_deps=[DIRNAME + "-controller-manager"])

    docker_build_with_restart(IMG, '.', 
     dockerfile_contents=DOCKERFILE,
     entrypoint='/manager',
     only=['./bin/manager'],
     live_update=[
           sync('./bin/manager', '/manager'),
       ]
    )
# deploy_cert_manager(version = "v1.1.0")
# capi()
create_config()
kubebuilder("schiff.telekom.de") 
