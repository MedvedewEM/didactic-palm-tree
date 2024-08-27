NC=$(tput setaf 0)
RED=$(tput setaf 1)
GREEN=$(tput setaf 2)
BLUE=$(tput setaf 4)

JITTER=${RANDOM}_

generate_test_files() {
    touch ${JITTER}0B.bin
    echo "a" > ${JITTER}2B.bin
    dd if=/dev/urandom of=${JITTER}10MB.bin bs=10M count=1
    dd if=/dev/urandom of=${JITTER}15MB.bin bs=15M count=1
    dd if=/dev/urandom of=${JITTER}50MB.bin bs=10M count=5
    dd if=/dev/urandom of=${JITTER}100MB.bin bs=100M count=1
    dd if=/dev/urandom of=${JITTER}300MB.bin bs=100M count=3
    dd if=/dev/urandom of=${JITTER}600MB.bin bs=100M count=6
    dd if=/dev/urandom of=${JITTER}1200MB.bin bs=100M count=12
}

equal() {
    if [[ "$1" == "$2" ]]
    then
        echo "${GREEN}OK${NC}"
    else
        echo "${RED}ERROR${NC}" $1 $2
    fi
}

upload() {
    echo $(curl -s -F input=@$1 "localhost:8080/upload")
}

download() {
    curl -s -o $1 "localhost:8080/download?uuid=$2"
}

UUIDs=()
filenames=()

script_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )

# generate files to upload
mkdir "$script_path/tmp"
cd "$script_path/tmp"
echo "${BLUE}generating test files${NC}"
generate_test_files
echo "${BLUE}generated test files${NC}"
echo;

input=(
    "${JITTER}0B.bin" "${JITTER}2B.bin" "${JITTER}10MB.bin" "${JITTER}15MB.bin" "${JITTER}50MB.bin" "${JITTER}100MB.bin" "${JITTER}300MB.bin" "${JITTER}600MB.bin" "${JITTER}1200MB.bin"
)

# upload files
for filename in ${input[*]};
do
    echo "${BLUE}uploading file: ${NC}" $filename
    uuid=$(upload $filename)
    echo "${BLUE}uploaded uuid: ${NC}" $uuid

    UUIDs+=($uuid)
    filenames+=($filename)
done
echo;

# download files
for i in $(seq 0 $(( ${#input[@]} - 1 )));
do
    echo "${BLUE}downloading file: ${NC}" ${filenames[$i]}
    download "d"${filenames[$i]} ${UUIDs[$i]}
    echo "${BLUE}downloaded file: ${NC}" "d"${filenames[$i]}
done
echo;

# check hashes of original and downloaded files
for filename in ${filenames[*]};
do
    original=$(shasum -a 256 $filename | cut -d ' ' -f1)
    downloaded=$(shasum -a 256 "d"$filename | cut -d ' ' -f1)

    equal $original $downloaded
done

# clean
rm -frd "$script_path/tmp"
