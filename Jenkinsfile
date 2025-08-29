pipeline {
    agent any
    
    environment {
        GO_VERSION = '1.21'
        DOCKER_REGISTRY = 'your-registry.com'
        IMAGE_NAME = 'migr8/migr8'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
                script {
                    env.GIT_COMMIT = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
                    env.BUILD_TIME = sh(returnStdout: true, script: 'date -u +%Y-%m-%dT%H:%M:%SZ').trim()
                }
            }
        }
        
        stage('Setup') {
            parallel {
                stage('Go Setup') {
                    steps {
                        sh '''
                            # Install Go if not present
                            if ! command -v go &> /dev/null; then
                                wget -O go.tar.gz https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
                                sudo tar -C /usr/local -xzf go.tar.gz
                                export PATH=$PATH:/usr/local/go/bin
                            fi
                            go version
                        '''
                    }
                }
                
                stage('Database Setup') {
                    steps {
                        script {
                            // Start test databases
                            sh '''
                                docker run -d --name postgres-test \
                                    -e POSTGRES_PASSWORD=testpass \
                                    -e POSTGRES_USER=testuser \
                                    -e POSTGRES_DB=migr8_test \
                                    -p 5432:5432 \
                                    postgres:15
                                    
                                docker run -d --name mysql-test \
                                    -e MYSQL_ROOT_PASSWORD=rootpass \
                                    -e MYSQL_DATABASE=migr8_test \
                                    -e MYSQL_USER=testuser \
                                    -e MYSQL_PASSWORD=testpass \
                                    -p 3306:3306 \
                                    mysql:8.0
                                
                                # Wait for databases to be ready
                                sleep 30
                            '''
                        }
                    }
                }
            }
        }
        
        stage('Dependencies') {
            steps {
                sh 'go mod download'
                sh 'go mod verify'
            }
        }
        
        stage('Code Quality') {
            parallel {
                stage('Lint') {
                    steps {
                        sh '''
                            # Install golangci-lint if not present
                            if ! command -v golangci-lint &> /dev/null; then
                                curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.0
                            fi
                            golangci-lint run --timeout=5m
                        '''
                    }
                    post {
                        always {
                            publishHTML([
                                allowMissing: false,
                                alwaysLinkToLastBuild: true,
                                keepAll: true,
                                reportDir: '.',
                                reportFiles: 'lint-report.xml',
                                reportName: 'Lint Report'
                            ])
                        }
                    }
                }
                
                stage('Security Scan') {
                    steps {
                        sh '''
                            # Install gosec if not present
                            if ! command -v gosec &> /dev/null; then
                                go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
                            fi
                            gosec -fmt json -out gosec-report.json ./...
                        '''
                    }
                    post {
                        always {
                            archiveArtifacts artifacts: 'gosec-report.json', allowEmptyArchive: true
                        }
                    }
                }
            }
        }
        
        stage('Test') {
            parallel {
                stage('Unit Tests') {
                    steps {
                        sh 'go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...'
                    }
                    post {
                        always {
                            // Publish test results
                            sh 'go test -v ./... 2>&1 | go-junit-report > test-report.xml || true'
                            junit 'test-report.xml'
                            
                            // Publish coverage
                            sh 'go tool cover -html=coverage.txt -o coverage.html'
                            publishHTML([
                                allowMissing: false,
                                alwaysLinkToLastBuild: true,
                                keepAll: true,
                                reportDir: '.',
                                reportFiles: 'coverage.html',
                                reportName: 'Coverage Report'
                            ])
                        }
                    }
                }
                
                stage('Integration Tests') {
                    steps {
                        sh '''
                            # PostgreSQL integration tests
                            export TEST_DB_DRIVER=postgres
                            export TEST_DB_HOST=localhost
                            export TEST_DB_PORT=5432
                            export TEST_DB_USER=testuser
                            export TEST_DB_PASS=testpass
                            export TEST_DB_NAME=migr8_test
                            go test -v ./pkg/database -tags=integration
                            
                            # MySQL integration tests
                            export TEST_DB_DRIVER=mysql
                            export TEST_DB_HOST=localhost
                            export TEST_DB_PORT=3306
                            export TEST_DB_USER=testuser
                            export TEST_DB_PASS=testpass
                            export TEST_DB_NAME=migr8_test
                            go test -v ./pkg/database -tags=integration
                        '''
                    }
                }
            }
        }
        
        stage('Build') {
            parallel {
                stage('Binary') {
                    steps {
                        sh '''
                            CGO_ENABLED=0 go build \
                                -ldflags "-X 'migr8/internal/cli.Version=${BUILD_NUMBER}' \
                                          -X 'migr8/internal/cli.GitCommit=${GIT_COMMIT}' \
                                          -X 'migr8/internal/cli.BuildTime=${BUILD_TIME}'" \
                                -o migr8 .
                        '''
                        
                        archiveArtifacts artifacts: 'migr8', fingerprint: true
                    }
                }
                
                stage('Docker Image') {
                    when { 
                        anyOf {
                            branch 'main'
                            branch 'develop'
                            buildingTag()
                        }
                    }
                    steps {
                        script {
                            def image = docker.build("${IMAGE_NAME}:${BUILD_NUMBER}")
                            
                            // Test the built image
                            image.inside('--network host') {
                                sh './migr8 version'
                            }
                            
                            // Push to registry
                            docker.withRegistry("https://${DOCKER_REGISTRY}", 'docker-registry-credentials') {
                                image.push()
                                image.push('latest')
                                
                                if (env.BRANCH_NAME == 'main') {
                                    image.push('stable')
                                }
                            }
                        }
                    }
                }
            }
        }
        
        stage('Deploy to Staging') {
            when { branch 'develop' }
            steps {
                sh '''
                    # Deploy to staging environment
                    echo "Deploying to staging..."
                    
                    # Example: Update staging deployment
                    kubectl set image deployment/migr8-staging migr8=${DOCKER_REGISTRY}/${IMAGE_NAME}:${BUILD_NUMBER}
                    kubectl rollout status deployment/migr8-staging
                    
                    # Run smoke tests
                    kubectl run migr8-smoke-test --image=${DOCKER_REGISTRY}/${IMAGE_NAME}:${BUILD_NUMBER} \
                        --rm -i --restart=Never -- version
                '''
            }
        }
        
        stage('Deploy to Production') {
            when { 
                allOf {
                    branch 'main'
                    not { changeRequest() }
                }
            }
            steps {
                script {
                    timeout(time: 5, unit: 'MINUTES') {
                        input message: 'Deploy to production?', ok: 'Deploy',
                              submitterParameter: 'APPROVER'
                    }
                }
                
                sh '''
                    echo "Deploying to production..."
                    echo "Approved by: ${APPROVER}"
                    
                    # Example: Update production deployment
                    kubectl set image deployment/migr8-production migr8=${DOCKER_REGISTRY}/${IMAGE_NAME}:${BUILD_NUMBER}
                    kubectl rollout status deployment/migr8-production
                    
                    # Run production smoke tests
                    kubectl run migr8-prod-smoke-test --image=${DOCKER_REGISTRY}/${IMAGE_NAME}:${BUILD_NUMBER} \
                        --rm -i --restart=Never -- version
                '''
            }
        }
    }
    
    post {
        always {
            // Cleanup test databases
            sh '''
                docker stop postgres-test mysql-test || true
                docker rm postgres-test mysql-test || true
            '''
            
            // Clean workspace
            cleanWs()
        }
        
        success {
            slackSend(
                channel: '#deployments',
                color: 'good',
                message: ":white_check_mark: *Migr8 Build Successful* - Build #${BUILD_NUMBER}\n" +
                        "Branch: ${BRANCH_NAME}\n" +
                        "Commit: ${GIT_COMMIT}\n" +
                        "Duration: ${currentBuild.durationString}"
            )
        }
        
        failure {
            slackSend(
                channel: '#deployments',
                color: 'danger',
                message: ":x: *Migr8 Build Failed* - Build #${BUILD_NUMBER}\n" +
                        "Branch: ${BRANCH_NAME}\n" +
                        "Commit: ${GIT_COMMIT}\n" +
                        "Duration: ${currentBuild.durationString}\n" +
                        "Check: ${BUILD_URL}"
            )
        }
    }
}