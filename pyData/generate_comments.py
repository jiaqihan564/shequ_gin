import random
import pymysql
from faker import Faker
from config import DB_CONFIG, COMMENT_COUNT, USER_COUNT, ARTICLE_COUNT, RESOURCE_COUNT

# 初始化Faker
fake = Faker('zh_CN')

# 评论状态
comment_statuses = [0, 1, 2]  # 0-已删除，1-正常，2-已折叠

def generate_comments():
    """生成评论数据"""
    # 连接数据库
    connection = pymysql.connect(**DB_CONFIG)
    try:
        with connection.cursor() as cursor:
            # 生成文章评论
            article_comment_count = int(COMMENT_COUNT * 0.7)  # 70%的文章评论
            resource_comment_count = COMMENT_COUNT - article_comment_count  # 30%的资源评论
            
            # 生成文章评论数据
            for i in range(article_comment_count):
                # 随机选择文章和用户
                article_id = random.randint(1, ARTICLE_COUNT)
                user_id = random.randint(1, USER_COUNT)
                
                # 90%的一级评论，10%的回复评论
                if random.random() > 0.9:
                    # 获取该文章已有的评论作为父评论
                    cursor.execute("""
                        SELECT id, user_id FROM article_comments 
                        WHERE article_id = %s AND parent_id = 0 AND status = 1
                        ORDER BY RAND() LIMIT 1
                    """, (article_id,))
                    parent_result = cursor.fetchone()
                    if parent_result:
                        parent_id = parent_result[0]
                        reply_to_user_id = parent_result[1]
                        root_id = parent_id  # 简化处理，实际应该查找父评论的root_id
                    else:
                        parent_id = 0
                        reply_to_user_id = None
                        root_id = 0
                else:
                    parent_id = 0
                    reply_to_user_id = None
                    root_id = 0
                
                content = fake.text(max_nb_chars=300)
                like_count = random.randint(0, 100)
                reply_count = random.randint(0, 20) if parent_id == 0 else 0  # 只有顶级评论有回复数
                status = random.choices(comment_statuses, weights=[3, 95, 2], k=1)[0]  # 3%已删除，95%正常，2%已折叠
                created_at = fake.date_time_between(start_date='-2y', end_date='now')
                updated_at = fake.date_time_between(start_date=created_at, end_date='now')
                
                # 插入文章评论数据
                article_comment_sql = """
                INSERT INTO article_comments (article_id, user_id, parent_id, root_id, reply_to_user_id,
                                            content, like_count, reply_count, status, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                cursor.execute(article_comment_sql, (
                    article_id, user_id, parent_id, root_id, reply_to_user_id,
                    content, like_count, reply_count, status, created_at, updated_at
                ))
                comment_id = cursor.lastrowid
                
                # 为评论生成点赞数据（30%的评论有点赞）
                if random.random() > 0.7:
                    like_count = random.randint(1, 50)
                    for _ in range(like_count):
                        like_user_id = random.randint(1, USER_COUNT)
                        
                        # 检查是否已点赞
                        cursor.execute("""
                            SELECT id FROM article_comment_likes 
                            WHERE comment_id = %s AND user_id = %s
                        """, (comment_id, like_user_id))
                        if not cursor.fetchone():
                            like_sql = """
                            INSERT INTO article_comment_likes (comment_id, user_id, created_at)
                            VALUES (%s, %s, %s)
                            """
                            cursor.execute(like_sql, (comment_id, like_user_id, fake.date_time_between(start_date=created_at, end_date='now')))
                
                # 每1000条提交一次
                if (i + 1) % 1000 == 0:
                    connection.commit()
                    print(f"已插入 {i + 1} 条文章评论数据")
            
            # 生成资源评论数据
            for i in range(resource_comment_count):
                # 随机选择资源和用户
                resource_id = random.randint(1, RESOURCE_COUNT)
                user_id = random.randint(1, USER_COUNT)
                
                # 90%的一级评论，10%的回复评论
                if random.random() > 0.9:
                    # 获取该资源已有的评论作为父评论
                    cursor.execute("""
                        SELECT id, user_id FROM resource_comments 
                        WHERE resource_id = %s AND parent_id = 0 AND status = 1
                        ORDER BY RAND() LIMIT 1
                    """, (resource_id,))
                    parent_result = cursor.fetchone()
                    if parent_result:
                        parent_id = parent_result[0]
                        reply_to_user_id = parent_result[1]
                        root_id = parent_id  # 简化处理
                    else:
                        parent_id = 0
                        reply_to_user_id = None
                        root_id = 0
                else:
                    parent_id = 0
                    reply_to_user_id = None
                    root_id = 0
                
                content = fake.text(max_nb_chars=300)
                like_count = random.randint(0, 100)
                reply_count = random.randint(0, 20) if parent_id == 0 else 0  # 只有顶级评论有回复数
                status = random.choices(comment_statuses, weights=[3, 95, 2], k=1)[0]  # 3%已删除，95%正常，2%已折叠
                created_at = fake.date_time_between(start_date='-2y', end_date='now')
                updated_at = fake.date_time_between(start_date=created_at, end_date='now')
                
                # 插入资源评论数据
                resource_comment_sql = """
                INSERT INTO resource_comments (resource_id, user_id, parent_id, root_id, reply_to_user_id,
                                             content, like_count, reply_count, status, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                cursor.execute(resource_comment_sql, (
                    resource_id, user_id, parent_id, root_id, reply_to_user_id,
                    content, like_count, reply_count, status, created_at, updated_at
                ))
                comment_id = cursor.lastrowid
                
                # 为评论生成点赞数据（30%的评论有点赞）
                if random.random() > 0.7:
                    like_count = random.randint(1, 50)
                    for _ in range(like_count):
                        like_user_id = random.randint(1, USER_COUNT)
                        
                        # 检查是否已点赞
                        cursor.execute("""
                            SELECT id FROM resource_comment_likes 
                            WHERE comment_id = %s AND user_id = %s
                        """, (comment_id, like_user_id))
                        if not cursor.fetchone():
                            like_sql = """
                            INSERT INTO resource_comment_likes (comment_id, user_id, created_at)
                            VALUES (%s, %s, %s)
                            """
                            cursor.execute(like_sql, (comment_id, like_user_id, fake.date_time_between(start_date=created_at, end_date='now')))
                
                # 每1000条提交一次
                if (i + 1) % 1000 == 0:
                    connection.commit()
                    print(f"已插入 {i + 1} 条资源评论数据")
            
            # 更新文章和资源的评论数
            print("正在更新文章和资源的评论数...")
            cursor.execute("""
                UPDATE articles a 
                SET comment_count = (
                    SELECT COUNT(*) FROM article_comments ac 
                    WHERE ac.article_id = a.id AND ac.status = 1
                )
            """)
            
            cursor.execute("""
                UPDATE resources r 
                SET comment_count = (
                    SELECT COUNT(*) FROM resource_comments rc 
                    WHERE rc.resource_id = r.id AND rc.status = 1
                )
            """)
            
            # 最后提交
            connection.commit()
            print(f"评论数据生成完成，共 {COMMENT_COUNT} 条记录")
            
    except Exception as e:
        print(f"生成评论数据时出错: {e}")
        connection.rollback()
    finally:
        connection.close()

if __name__ == "__main__":
    generate_comments()