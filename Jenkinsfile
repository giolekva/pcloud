pipeline {
    agent {
        kubernetes {
			yaml '''
                apiVersion: v1
                kind: Pod
                spec:
                  containers:
                  - name: golang
                    image: golang:1.22.2-alpine3.19
                    tty: true
            '''
		}
    }
    stages {
        stage('installer auth') {
            steps {
				container('golang') {
                	dir('core/installer') {
                		sh 'go mod tidy'
                		sh 'go build cmd/*.go'
                		sh 'go test ./...'
					}
                    dir('core/auth/memberships') {
                        sh 'go mod tidy'
                		sh 'go build *.go'
                		sh 'go test ./...'
					}
				}
            }
        }
    }
	post {
        success {
            gerritReview labels: [Verified: 1], message: env.BUILD_URL
        }
        unstable {
		    gerritReview labels: [Verified: 0], message: env.BUILD_URL
		}
        failure {
            gerritReview labels: [Verified: -1], message: env.BUILD_URL
        }
    }
}