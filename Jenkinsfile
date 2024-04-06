pipeline {
    agent { docker { image 'golang:1.22.2-alpine3.19' } }
    stages {
        stage('build') {
            steps {
                sh 'go version'
            }
        }
    }
}