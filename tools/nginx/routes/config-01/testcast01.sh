# 首先先检查使用前缀字符定义的location，选择最长匹配的项并记录下来。
# 如果找到了精确匹配的location，也就是使用了=修饰符的location，结束查找，使用它的配置。
# 然后按顺序查找使用正则定义的location，如果匹配则停止查找，使用它定义的配置。
# 如果没有匹配的正则location，则使用前面记录的最长匹配前缀字符location。
#
# 1. 精确匹配
# 2. 正则匹配,停止查找
# 3. 最长匹配前缀字符location
curl http://localhost:8080
curl http://localhost:8080/ws
curl http://localhost:8080/api/v3/login
curl http://localhost:8080/static/login.js
curl http://localhost:8080/ws/chat.css
