# setting up colors
BLU='\033[0;34m'
GRN='\033[0;32m'
LBLU='\033[0;94m'
NOC='\033[0m' # No Color

echo_info(){
    printf "\n${LBLU}%s\n${NOC}" "$1"
}

echo_step(){
    printf "\n\n${BLU}>>>>>>> %s${NOC}\n" "$1"
}

echo_sub_step(){
    printf "\n${BLU}>>> %s${NOC}\n" "$1"
}

echo_step_completed(){
    printf "${GRN} [✔]${NOC}"
}

echo_success(){
    printf "\n${GRN}%s${NOC}\n" "$1"
}
