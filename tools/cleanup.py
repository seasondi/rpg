import json
import psutil
from pymongo import MongoClient

def check_server_running():
    print("checking server is running...")
    processes = ["game.exe", "db.exe"]
    for proc in psutil.process_iter(['pid', 'name']):
        try:
            if proc.info['name'] in processes:
                return True
        except (psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
            pass
    return False


if __name__ == "__main__":
    if check_server_running():
        print("============local server is running, please stop first============")
        exit(0)

    with open("./config/config.json", 'r', encoding='utf-8') as file:
        data = json.load(file)
        db_name = str(data["serverId"])
        uri = data["mongo"]["uri"]
        client = MongoClient(uri)
        db = client[db_name]
        collections = db.list_collection_names()
        for col in collections:
            print(f"cleanup {db_name}.{col} start")
            db[col].drop()
            print(f"cleanup {db_name}.[{col}] successfully")
        client.close()
        print("================clean database successfully=================")