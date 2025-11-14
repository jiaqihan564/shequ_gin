import random
import hashlib
import datetime
import pymysql
from faker import Faker
from config import DB_CONFIG, USER_COUNT

# 初始化Faker
fake = Faker('zh_CN')

# 性别选项
genders = [0, 1, 2]  # 0-未知，1-男，2-女

# 角色选项
roles = ['user', 'admin']

# 账户状态选项
account_statuses = [0, 1, 2]  # 0-禁用，1-正常，2-锁定

# 省份城市数据
provinces_cities = {
    '北京市': ['东城区', '西城区', '朝阳区', '丰台区', '石景山区'],
    '上海市': ['黄浦区', '徐汇区', '长宁区', '静安区', '普陀区'],
    '广东省': ['广州市', '深圳市', '珠海市', '汕头市', '佛山市'],
    '江苏省': ['南京市', '无锡市', '徐州市', '常州市', '苏州市'],
    '浙江省': ['杭州市', '宁波市', '温州市', '嘉兴市', '湖州市'],
    '山东省': ['济南市', '青岛市', '淄博市', '枣庄市', '东营市'],
    '河南省': ['郑州市', '开封市', '洛阳市', '平顶山市', '安阳市'],
    '河北省': ['石家庄市', '唐山市', '秦皇岛市', '邯郸市', '邢台市'],
    '四川省': ['成都市', '自贡市', '攀枝花市', '泸州市', '德阳市'],
    '湖南省': ['长沙市', '株洲市', '湘潭市', '衡阳市', '邵阳市']
}

def hash_password(password):
    """生成密码哈希"""
    return hashlib.sha256(password.encode('utf-8')).hexdigest()

def generate_users():
    """生成用户数据"""
    # 连接数据库
    connection = pymysql.connect(**DB_CONFIG)
    try:
        with connection.cursor() as cursor:
            # 生成用户认证数据
            user_auth_data = []
            user_profile_data = []
            
            for i in range(USER_COUNT):
                # 用户名和邮箱
                username = fake.user_name() + str(i)
                email = fake.email()
                password_hash = hash_password('123456')  # 默认密码
                role = random.choices(roles, weights=[95, 5], k=1)[0]  # 95%普通用户，5%管理员
                auth_status = 1  # 默认已认证
                account_status = random.choices(account_statuses, weights=[5, 90, 5], k=1)[0]  # 5%禁用，90%正常，5%锁定
                last_login_time = fake.date_time_between(start_date='-1y', end_date='now') if random.random() > 0.3 else None
                last_login_ip = fake.ipv4() if last_login_time else None
                failed_login_count = random.randint(0, 10)
                created_at = fake.date_time_between(start_date='-2y', end_date='-1d')
                updated_at = fake.date_time_between(start_date=created_at, end_date='now')
                
                # 插入用户认证数据
                user_auth_sql = """
                INSERT INTO user_auth (username, password_hash, email, role, auth_status, account_status, 
                                      last_login_time, last_login_ip, failed_login_count, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                cursor.execute(user_auth_sql, (
                    username, password_hash, email, role, auth_status, account_status,
                    last_login_time, last_login_ip, failed_login_count, created_at, updated_at
                ))
                user_id = cursor.lastrowid
                
                # 生成用户资料数据
                nickname = fake.name()
                bio = fake.text(max_nb_chars=200) if random.random() > 0.5 else None
                avatar_url = fake.image_url() if random.random() > 0.3 else None
                phone = fake.phone_number() if random.random() > 0.4 else None
                gender = random.choice(genders)
                birthday = fake.date_of_birth(minimum_age=18, maximum_age=60) if random.random() > 0.3 else None
                
                # 随机选择省份和城市
                province = random.choice(list(provinces_cities.keys()))
                city = random.choice(provinces_cities[province])
                
                website = fake.url() if random.random() > 0.7 else None
                github = fake.user_name() if random.random() > 0.6 else None
                created_at_profile = created_at
                updated_at_profile = fake.date_time_between(start_date=created_at_profile, end_date='now')
                
                # 插入用户资料数据
                user_profile_sql = """
                INSERT INTO user_profile (user_id, nickname, bio, avatar_url, phone, gender, birthday, 
                                         province, city, website, github, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                cursor.execute(user_profile_sql, (
                    user_id, nickname, bio, avatar_url, phone, gender, birthday,
                    province, city, website, github, created_at_profile, updated_at_profile
                ))
                
                # 每1000条提交一次
                if (i + 1) % 1000 == 0:
                    connection.commit()
                    print(f"已插入 {i + 1} 条用户数据")
            
            # 最后提交
            connection.commit()
            print(f"用户数据生成完成，共 {USER_COUNT} 条记录")
            
    except Exception as e:
        print(f"生成用户数据时出错: {e}")
        connection.rollback()
    finally:
        connection.close()

if __name__ == "__main__":
    generate_users()