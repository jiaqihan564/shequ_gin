import random
import pymysql
from faker import Faker
from config import DB_CONFIG, CHAT_MESSAGE_COUNT, USER_COUNT

# 初始化Faker
fake = Faker('zh_CN')

# 消息类型
message_types = [1, 2]  # 1-普通消息，2-系统消息

# 消息状态
message_statuses = [0, 1]  # 0-已删除，1-正常

def generate_chat_messages():
    """生成聊天消息数据"""
    # 连接数据库
    connection = pymysql.connect(**DB_CONFIG)
    try:
        with connection.cursor() as cursor:
            # 获取一部分用户信息用于聊天消息
            cursor.execute("SELECT id, username FROM user_auth LIMIT 10000")
            users = cursor.fetchall()
            
            if not users:
                print("没有找到用户数据，请先生成用户数据")
                return
            
            # 生成聊天消息数据
            for i in range(CHAT_MESSAGE_COUNT):
                # 随机选择用户
                user = random.choice(users)
                user_id = user[0]
                username = user[1]
                
                # 获取用户昵称和头像
                cursor.execute("SELECT nickname, avatar_url FROM user_profile WHERE user_id = %s", (user_id,))
                profile_result = cursor.fetchone()
                nickname = profile_result[0] if profile_result and profile_result[0] else username
                avatar = profile_result[1] if profile_result and profile_result[1] else None
                
                content = fake.sentence(nb_words=20)
                message_type = random.choices(message_types, weights=[95, 5], k=1)[0]  # 95%普通消息，5%系统消息
                send_time = fake.date_time_between(start_date='-1y', end_date='now')
                ip_address = fake.ipv4()
                status = random.choices(message_statuses, weights=[5, 95], k=1)[0]  # 5%已删除，95%正常
                created_at = send_time
                
                # 插入聊天消息数据
                chat_message_sql = """
                INSERT INTO chat_messages (user_id, username, nickname, avatar, content, 
                                         message_type, send_time, ip_address, status, created_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                cursor.execute(chat_message_sql, (
                    user_id, username, nickname, avatar, content,
                    message_type, send_time, ip_address, status, created_at
                ))
                
                # 每1000条提交一次
                if (i + 1) % 1000 == 0:
                    connection.commit()
                    print(f"已插入 {i + 1} 条聊天消息数据")
            
            # 最后提交
            connection.commit()
            print(f"聊天消息数据生成完成，共 {CHAT_MESSAGE_COUNT} 条记录")
            
    except Exception as e:
        print(f"生成聊天消息数据时出错: {e}")
        connection.rollback()
    finally:
        connection.close()

if __name__ == "__main__":
    generate_chat_messages()