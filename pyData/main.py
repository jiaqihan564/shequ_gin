"""
生成测试数据的主程序
按顺序执行各个数据生成脚本
"""

import time
import subprocess
import sys
import os

def run_script(script_name):
    """运行指定的Python脚本"""
    print(f"\n开始执行 {script_name}...")
    start_time = time.time()
    
    try:
        # 获取当前脚本所在目录
        current_dir = os.path.dirname(os.path.abspath(__file__))
        # 使用 errors='replace' 处理编码问题
        result = subprocess.run([sys.executable, script_name], 
                              capture_output=True, text=True, 
                              cwd=current_dir, encoding='utf-8', errors='replace')
        if result.returncode == 0:
            print(f"✓ {script_name} 执行成功")
            if result.stdout:
                print(result.stdout)
        else:
            print(f"✗ {script_name} 执行失败")
            if result.stderr:
                print(result.stderr)
            return False
    except Exception as e:
        print(f"✗ 运行 {script_name} 时出错: {e}")
        return False
    
    end_time = time.time()
    print(f"执行时间: {end_time - start_time:.2f} 秒")
    return True

def main():
    """主函数"""
    print("开始生成测试数据...")
    start_time = time.time()
    
    # 确保在正确的目录中执行
    current_dir = os.path.dirname(os.path.abspath(__file__))
    os.chdir(current_dir)
    
    # 按顺序执行数据生成脚本
    scripts = [
        'generate_users.py',           # 生成用户数据
        'generate_articles.py',        # 生成文章数据
        'generate_resources.py',       # 生成资源数据
        'generate_comments.py',        # 生成评论数据
        'generate_chat_messages.py',   # 生成聊天消息数据
        'generate_likes.py',           # 生成点赞数据
        'generate_login_history.py',   # 生成登录历史数据
        'generate_statistics.py'       # 生成统计数据
    ]
    
    success_count = 0
    for script in scripts:
        if run_script(script):
            success_count += 1
        else:
            print(f"执行 {script} 失败，停止后续脚本执行")
            break
    
    end_time = time.time()
    
    print(f"\n=== 数据生成完成 ===")
    print(f"成功执行脚本数: {success_count}/{len(scripts)}")
    print(f"总耗时: {end_time - start_time:.2f} 秒")
    
    if success_count == len(scripts):
        print("🎉 所有数据生成完成！")
    else:
        print("⚠️  部分数据生成失败，请检查错误信息")

if __name__ == "__main__":
    main()