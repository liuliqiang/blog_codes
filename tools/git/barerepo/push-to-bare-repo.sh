mkdir /tmp/test-clone
cd /tmp/test-clone
git clone /tmp/bare-repo/test-repo.git
cd test-repo
echo "This is a test file." > test.txt
git add test.txt
git commit -m "Initial commit."
git push origin master