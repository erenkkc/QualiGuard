import os
import sys

DB_PASSWORD = "super-secret-password-123"


def risky(user_id):
    unused_result = 42
    try:
        raise ValueError("boom")
    except:
        pass

    try:
        raise RuntimeError("fail")
    except ValueError:
        pass

    eval("print('hello')")
    cursor = None
    cursor.execute(f"SELECT * FROM users WHERE id = {user_id}")


def very_complex(value):
    if value > 0:
        if value > 1:
            if value > 2:
                if value > 3:
                    if value > 4:
                        if value > 5:
                            if value > 6:
                                if value > 7:
                                    if value > 8:
                                        if value > 9:
                                            if value > 10:
                                                return "complex"
    return "simple"
