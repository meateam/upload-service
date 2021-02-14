//upload services meateam
pipeline {
  agent {    
       kubernetes {
       defaultContainer 'dind-slave'  
       yaml """
      apiVersion: v1 
      kind: Pod 
      metadata: 
          name: k8s-worker
      spec: 
          containers: 
            - name: dind-slave
              image: aymdev/dind-compose
              resources: 
                  requests: 
                      cpu: 20m 
                      memory: 512Mi 
              securityContext: 
                  privileged: true 
              volumeMounts: 
                - name: docker-graph-storage 
                  mountPath: /var/lib/docker 
          volumes: 
            - name: docker-graph-storage 
              emptyDir: {}
 """
    }
  }
    stages {
      // this stage create enviroment variable from git for discored massage
      stage('get_commit_msg') {
        steps {
          container('jnlp'){
          script {
            env.GIT_COMMIT_MSG = sh (script: 'git log -1 --pretty=%B ${GIT_COMMIT}', returnStdout: true).trim()
            env.GIT_SHORT_COMMIT = sh(returnStdout: true, script: "git log -n 1 --pretty=format:'%h'").trim()
            env.GIT_COMMITTER_EMAIL = sh (script: "git --no-pager show -s --format='%ae'", returnStdout: true  ).trim()
            env.GIT_REPO_NAME = scm.getUserRemoteConfigs()[0].getUrl().tokenize('/')[3].split("\\.")[0]
           
            // Takes the branch name and replaces the slashes with the %2F mark 
            env.BRANCH_FOR_URL = sh([script: "echo ${GIT_BRANCH} | sed 's;/;%2F;g'", returnStdout: true]).trim()
            // Takes the job path variable and replaces the slashes with the %2F mark 
            env.JOB_PATH = sh([script: "echo ${JOB_NAME} | sed 's;/;%2F;g'", returnStdout: true]).trim()
            // creating variable that contain the job path without the branch name  
            env.JOB_WITHOUT_BRANCH = sh([script: "echo ${env.JOB_PATH} | sed 's;${BRANCH_FOR_URL};'';g'", returnStdout: true]).trim() 
            //  creating variable that contain the JOB_WITHOUT_BRANCH variable without the last 3 characters 
            env.JOB_FOR_URL = sh([script: "echo ${JOB_WITHOUT_BRANCH}|rev | cut -c 4- | rev", returnStdout: true]).trim()  
            echo "${env.JOB_FOR_URL}" 
          }
        }
      }
    }
      // run unit test using docker-compose with mongo 
      stage('build dockerfile of tests') {
          steps {
            sh 'docker-compose -f docker-compose.test.yml up --exit-code-from upload_service_test'
          }
          post {
            always {
              discordSend description: '**service**: '+ env.GIT_REPO_NAME + '\n **Build**:' + " " + env.BUILD_NUMBER + '\n **Branch**:' + " " + env.GIT_BRANCH + '\n **Status**:' + " " +  currentBuild.result + '\n \n \n **Commit ID**:'+ " " + env.GIT_SHORT_COMMIT + '\n **commit massage**:' + " " + env.GIT_COMMIT_MSG + '\n **commit email**:' + " " + env.GIT_COMMITTER_EMAIL, footer: '', image: '', link: 'http://jnk-devops-ci-cd.northeurope.cloudapp.azure.com/blue/organizations/jenkins/'+env.JOB_FOR_URL+'/detail/'+env.BRANCH_FOR_URL+'/'+env.BUILD_NUMBER+'/pipeline', result: currentBuild.result, thumbnail: '', title: 'link to logs of unit test', webhookURL: env.discord    
            }
         }
      }
      // login to acr when pushed to branch master or develop 
      stage('login to azure container registry') {
          when {
            anyOf {
              branch 'master'; branch 'develop'
            }
          } 
          steps{ 
            withCredentials([usernamePassword(credentialsId:'DRIVE_ACR',usernameVariable: 'USER', passwordVariable: 'PASS')]) {
              sh "docker login  drivehub.azurecr.io -u ${USER} -p ${PASS}"
            }
          }
        } 
        //when pushed to master or develop build image and push to acr  
        stage('build dockerfile of system only for master and develop') {
            when {
              anyOf {
                 branch 'master'; branch 'develop'
              }
            }
            steps {
              script{
                if(env.GIT_BRANCH == 'master') {
                  sh "docker build -t  drivehub.azurecr.io/meateam/${env.GIT_REPO_NAME}:master ."
                  sh "docker push  drivehub.azurecr.io/meateam/${env.GIT_REPO_NAME}:master"
                }
                else if(env.GIT_BRANCH == 'develop') {
                  sh "docker build -t  drivehub.azurecr.io/meateam/${env.GIT_REPO_NAME}:develop ."
                  sh "docker push  drivehub.azurecr.io/meateam/${env.GIT_REPO_NAME}:develop"  
                }
              }  
            }
            post {
              always {
                discordSend description: '**service**: '+ env.GIT_REPO_NAME + '\n **Build**:' + " " + env.BUILD_NUMBER + '\n **Branch**:' + " " + env.GIT_BRANCH + '\n **Status**:' + " " +  currentBuild.result + '\n \n \n **Commit ID**:'+ " " + env.GIT_SHORT_COMMIT + '\n **commit massage**:' + " " + env.GIT_COMMIT_MSG + '\n **commit email**:' + " " + env.GIT_COMMITTER_EMAIL, footer: '', image: '', link: 'http://jnk-devops-ci-cd.northeurope.cloudapp.azure.com/blue/organizations/jenkins/'+env.JOB_FOR_URL+'/detail/'+env.BRANCH_FOR_URL+'/'+env.BUILD_NUMBER+'/pipeline', result: currentBuild.result, thumbnail: '', title: 'Logs build dockerfile master/develop', webhookURL: env.discord     
              }
            }
        }      
    }   
}
