import random
import pymysql
from faker import Faker
from config import DB_CONFIG, RESOURCE_COUNT, USER_COUNT

# 初始化Faker
fake = Faker('zh_CN')

# 资源状态
resource_statuses = [0, 1, 2]  # 0-已删除，1-正常，2-审核中

# 文件扩展名
file_extensions = ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'zip', 'rar', 'jpg', 'png', 'gif', 'mp4', 'mp3']

# 文件类型（MIME类型）
file_types = {
    'pdf': 'application/pdf',
    'doc': 'application/msword',
    'docx': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    'xls': 'application/vnd.ms-excel',
    'xlsx': 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    'ppt': 'application/vnd.ms-powerpoint',
    'pptx': 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    'zip': 'application/zip',
    'rar': 'application/x-rar-compressed',
    'jpg': 'image/jpeg',
    'png': 'image/png',
    'gif': 'image/gif',
    'mp4': 'video/mp4',
    'mp3': 'audio/mpeg'
}

def generate_resources():
    """生成资源数据"""
    # 连接数据库
    connection = pymysql.connect(**DB_CONFIG)
    try:
        with connection.cursor() as cursor:
            # 获取资源分类
            cursor.execute("SELECT id FROM resource_categories")
            category_ids = [row[0] for row in cursor.fetchall()]
            
            # 生成资源数据
            for i in range(RESOURCE_COUNT):
                # 随机选择上传者
                user_id = random.randint(1, USER_COUNT)
                
                # 资源信息
                title = fake.sentence(nb_words=8)
                description = fake.text(max_nb_chars=300)
                document = fake.text(max_nb_chars=1000) if random.random() > 0.5 else None
                category_id = random.choice(category_ids) if category_ids and random.random() > 0.2 else None
                file_name = fake.file_name()
                file_size = random.randint(1024, 1024*1024*100)  # 1KB到100MB
                extension = random.choice(file_extensions)
                file_type = file_types.get(extension, 'application/octet-stream')
                file_hash = fake.sha256()
                storage_path = f"/resources/{fake.date(pattern='%Y/%m/%d')}/{file_hash}.{extension}"
                total_chunks = 0 if random.random() > 0.1 else random.randint(2, 10)
                download_count = random.randint(0, 2000)
                view_count = random.randint(0, 3000)
                like_count = random.randint(0, 500)
                comment_count = random.randint(0, 200)
                status = random.choices(resource_statuses, weights=[2, 95, 3], k=1)[0]  # 2%已删除，95%正常，3%审核中
                created_at = fake.date_time_between(start_date='-2y', end_date='now')
                updated_at = fake.date_time_between(start_date=created_at, end_date='now')
                
                # 插入资源数据
                resource_sql = """
                INSERT INTO resources (user_id, title, description, document, category_id, file_name, file_size,
                                    file_type, file_extension, file_hash, storage_path, total_chunks,
                                    download_count, view_count, like_count, comment_count, status, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                cursor.execute(resource_sql, (
                    user_id, title, description, document, category_id, file_name, file_size,
                    file_type, extension, file_hash, storage_path, total_chunks,
                    download_count, view_count, like_count, comment_count, status, created_at, updated_at
                ))
                resource_id = cursor.lastrowid
                
                # 生成资源图片（60%的资源有图片）
                if random.random() > 0.4:
                    image_count = random.randint(1, 5)
                    for j in range(image_count):
                        image_url = fake.image_url()
                        image_order = j
                        is_cover = 1 if j == 0 else 0  # 第一张图片作为封面
                        
                        image_sql = """
                        INSERT INTO resource_images (resource_id, image_url, image_order, is_cover, created_at)
                        VALUES (%s, %s, %s, %s, %s)
                        """
                        cursor.execute(image_sql, (resource_id, image_url, image_order, is_cover, created_at))
                
                # 生成资源标签（每资源1-4个标签）
                tag_count = random.randint(1, 4)
                for j in range(tag_count):
                    tag_name = fake.word()
                    
                    tag_sql = """
                    INSERT INTO resource_tags (resource_id, tag_name, created_at)
                    VALUES (%s, %s, %s)
                    """
                    cursor.execute(tag_sql, (resource_id, tag_name, created_at))
                
                # 更新分类资源数
                if category_id:
                    cursor.execute("""
                        UPDATE resource_categories 
                        SET resource_count = resource_count + 1 
                        WHERE id = %s
                    """, (category_id,))
                
                # 每1000条提交一次
                if (i + 1) % 1000 == 0:
                    connection.commit()
                    print(f"已插入 {i + 1} 条资源数据")
            
            # 最后提交
            connection.commit()
            print(f"资源数据生成完成，共 {RESOURCE_COUNT} 条记录")
            
    except Exception as e:
        print(f"生成资源数据时出错: {e}")
        connection.rollback()
    finally:
        connection.close()

if __name__ == "__main__":
    generate_resources()