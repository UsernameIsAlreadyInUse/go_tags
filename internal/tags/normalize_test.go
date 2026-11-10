package tags

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"уже нормальный", "платье", "платье"},
		{"верхний регистр, кириллица", "ПЛАТЬЕ", "платье"},
		{"смешанный регистр, латиница", "PowerBank", "powerbank"},
		{"пробелы по краям", "  платье  ", "платье"},
		{"двойные пробелы внутри", "красное   платье", "красное платье"},
		{"табы и переводы строк", "красное\t\nплатье", "красное платье"},
		{"всё сразу", "  Красное   ПЛАТЬЕ ", "красное платье"},
		{"пустая строка", "", ""},
		{"только пробелы", "   \t ", ""},
		// Неразрывный пробел (U+00A0) не считается пробелом ни в Go (\s), ни в ClickHouse (RE2).
		// Тест фиксирует это решение, чтобы SQL и Go не разошлись.
		{"неразрывный пробел не трогаем", "a\u00a0b", "a\u00a0b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Normalize(tt.in); got != tt.want {
				t.Errorf("Normalize(%q) = %q, ожидали %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalize_Idempotent(t *testing.T) {
	// Нормализация уже нормализованной строки ничего не меняет.
	in := "  Красное   ПЛАТЬЕ "
	once := Normalize(in)
	if twice := Normalize(once); twice != once {
		t.Errorf("повторная нормализация изменила результат: %q -> %q", once, twice)
	}
}
