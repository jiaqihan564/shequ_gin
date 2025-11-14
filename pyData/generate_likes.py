import random
import pymysql
from faker import Faker
from config import DB_CONFIG, USER_COUNT, ARTICLE_COUNT, RESOURCE_COUNT

# 初始化Faker
fake = Faker()

def generate_likes():
    """生成点赞数据"""
    # 连接数据库
    connection = pymysql.connect(**DB_CONFIG)
    try:
        with connection.cursor() as cursor:
            # 生成文章点赞数据
            print("正在生成文章点赞数据...")
            article_like_count = 0
            
            # 为每篇文章生成点赞数据
            for article_id in range(1, ARTICLE_COUNT + 1):
                # 随机生成该文章的点赞数量（0-200）
                like_count = random.randint(0, 200)
                
                # 为文章生成点赞记录
                for _ in range(like_count):
                    user_id = random.randint(1, USER_COUNT)
                    created_at = fake.date_time_between(start_date='-1y', end_date='now')
                    
                    # 检查是否已点赞
                    cursor.execute("""
                        SELECT id FROM article_likes 
                        WHERE article_id = %s AND user_id = %s
                    """, (article_id, user_id))
                    
                    if not cursor.fetchone():
                        like_sql = """
                        INSERT INTO article_likes (article_id, user_id, created_at)
                        VALUES (%s, %s, %s)
                        """
                        cursor.execute(like_sql, (article_id, user_id, created_at))
                        article_like_count += 1
                
                # 每1000篇文章提交一次
                if article_id % 1000 == 0:
                    connection.commit()
                    print(f"已处理 {article_id} 篇文章的点赞数据")
            
            # 更新文章点赞数
            print("正在更新文章点赞数...")
            cursor.execute("""
                UPDATE articles a 
                SET like_count = (
                    SELECT COUNT(*) FROM article_likes al 
                    WHERE al.article_id = a.id
                )
            """)
            
            connection.commit()
            print(f"文章点赞数据生成完成，共 {article_like_count} 条记录")
            
            # 生成资源点赞数据
            print("正在生成资源点赞数据...")
            resource_like_count = 0
            
            # 为每个资源生成点赞数据
            for resource_id in range(1, RESOURCE_COUNT + 1):
                # 随机生成该资源的点赞数量（0-100）
                like_count = random.randint(0, 100)
                
                # 为资源生成点赞记录
                for _ in range(like_count):
                    user_id = random.randint(1, USER_COUNT)
                    created_at = fake.date_time_between(start_date='-1y', end_date='now')
                    
                    # 检查是否已点赞
                    cursor.execute("""
                        SELECT id FROM resource_likes 
                        WHERE resource_id = %s AND user_id = %s
                    """, (resource_id, user_id))
                    
                    if not cursor.fetchone():
                        like_sql = """
                        INSERT INTO resource_likes (resource_id, user_id, created_at)
                        VALUES (%s, %s, %s)
                        """
                        cursor.execute(like_sql, (resource_id, user_id, created_at))
                        resource_like_count += 1
                
                # 每1000个资源提交一次
                if resource_id % 1000 == 0:
                    connection.commit()
                    print(f"已处理 {resource_id} 个资源的点赞数据")
            
            # 更新资源点赞数
            print("正在更新资源点赞数...")
            cursor.execute("""
                UPDATE resources r 
                SET like_count = (
                    SELECT COUNT(*) FROM resource_likes rl 
                    WHERE rl.resource_id = r.id
                )
            """)
            
            # 最后提交
            connection.commit()
            print(f"资源点赞数据生成完成，共 {resource_like_count} 条记录")
            print(f"点赞数据生成完成，总共 {article_like_count + resource_like_count} 条记录")
            
    except Exception as e:
        print(f"生成点赞数据时出错: {e}")
        connection.rollback()
    finally:
        connection.close()

if __name__ == "__main__":
    generate_likes()