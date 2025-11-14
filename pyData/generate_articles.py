import random
import pymysql
from faker import Faker
from config import DB_CONFIG, ARTICLE_COUNT, USER_COUNT

# 初始化Faker
fake = Faker('zh_CN')

# 文章状态
article_statuses = [0, 1, 2]  # 0-草稿，1-已发布，2-已删除

# 编程语言
languages = ['python', 'javascript', 'java', 'go', 'cpp', 'rust', 'php', 'ruby', 'swift', 'kotlin']

def generate_articles():
    """生成文章数据"""
    # 连接数据库
    connection = pymysql.connect(**DB_CONFIG)
    try:
        with connection.cursor() as cursor:
            # 获取文章分类
            cursor.execute("SELECT id FROM article_categories")
            category_ids = [row[0] for row in cursor.fetchall()]
            
            # 获取文章标签
            cursor.execute("SELECT id FROM article_tags")
            tag_ids = [row[0] for row in cursor.fetchall()]
            
            # 生成文章数据
            for i in range(ARTICLE_COUNT):
                # 随机选择作者
                user_id = random.randint(1, USER_COUNT)
                
                # 文章标题和内容
                title = fake.sentence(nb_words=10)
                description = fake.text(max_nb_chars=200)
                content = fake.text(max_nb_chars=2000)
                status = random.choices(article_statuses, weights=[5, 90, 5], k=1)[0]  # 5%草稿，90%已发布，5%已删除
                view_count = random.randint(0, 5000)
                like_count = random.randint(0, 1000)
                comment_count = random.randint(0, 500)
                created_at = fake.date_time_between(start_date='-2y', end_date='now')
                updated_at = fake.date_time_between(start_date=created_at, end_date='now')
                
                # 插入文章数据
                article_sql = """
                INSERT INTO articles (user_id, title, description, content, status, view_count, 
                                    like_count, comment_count, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                cursor.execute(article_sql, (
                    user_id, title, description, content, status, view_count,
                    like_count, comment_count, created_at, updated_at
                ))
                article_id = cursor.lastrowid
                
                # 生成文章代码块（30%的文章有代码块）
                if random.random() > 0.7:
                    code_block_count = random.randint(1, 5)
                    for j in range(code_block_count):
                        language = random.choice(languages)
                        code_content = fake.text(max_nb_chars=500)
                        code_description = fake.sentence(nb_words=6) if random.random() > 0.5 else None
                        order_index = j
                        
                        code_block_sql = """
                        INSERT INTO article_code_blocks (article_id, language, code_content, description, order_index, created_at)
                        VALUES (%s, %s, %s, %s, %s, %s)
                        """
                        cursor.execute(code_block_sql, (
                            article_id, language, code_content, code_description, order_index, created_at
                        ))
                
                # 关联文章分类（每篇文章1-3个分类）
                article_categories = random.sample(category_ids, random.randint(1, min(3, len(category_ids))))
                for category_id in article_categories:
                    category_relation_sql = """
                    INSERT INTO article_category_relations (article_id, category_id, created_at)
                    VALUES (%s, %s, %s)
                    """
                    cursor.execute(category_relation_sql, (article_id, category_id, created_at))
                    
                    # 更新分类文章数
                    cursor.execute("""
                        UPDATE article_categories 
                        SET article_count = article_count + 1 
                        WHERE id = %s
                    """, (category_id,))
                
                # 关联文章标签（每篇文章1-5个标签）
                article_tags = random.sample(tag_ids, random.randint(1, min(5, len(tag_ids))))
                for tag_id in article_tags:
                    tag_relation_sql = """
                    INSERT INTO article_tag_relations (article_id, tag_id, created_at)
                    VALUES (%s, %s, %s)
                    """
                    cursor.execute(tag_relation_sql, (article_id, tag_id, created_at))
                    
                    # 更新标签文章数
                    cursor.execute("""
                        UPDATE article_tags 
                        SET article_count = article_count + 1 
                        WHERE id = %s
                    """, (tag_id,))
                
                # 每1000条提交一次
                if (i + 1) % 1000 == 0:
                    connection.commit()
                    print(f"已插入 {i + 1} 条文章数据")
            
            # 最后提交
            connection.commit()
            print(f"文章数据生成完成，共 {ARTICLE_COUNT} 条记录")
            
    except Exception as e:
        print(f"生成文章数据时出错: {e}")
        connection.rollback()
    finally:
        connection.close()

if __name__ == "__main__":
    generate_articles()