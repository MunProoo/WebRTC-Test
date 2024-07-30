// 카카오톡 채팅 만들기
function readValue(){
    
    // 채팅 입력에 사용되는 요소 모두 얻어오기
    const bg = document.getElementById("chatting-bg");

    const input = document.querySelector("#chatting-input");

    // input에 입력된 값이 있을 경우
    if(input.value.trim().length > 0){

        // 문자열.trim() : 문자열 양 끝에 공백을 모두 제거
        // ex) "           k        h            ".trim() -> "k        h"

        // input에 입력된 값을 얻어와 bg에 추가(누적)
        bg.innerHTML += "<p> <span>" + input.value + "</span></p>";

        // 요소.scrollTop        : 요소 내부 현재 스크롤 위치 반환
        // 요소.scrollTop = 위치 : 스크롤을 특정 위치 이동
        // 요소.scrollHeight     : 스크롤 전체 높이

        // bg의 스크롤을 제일 밑으로 내리기
        bg.scrollTop = bg.scrollHeight;
    }

    // input에 작성된 값 변경하기
    input.value = ""; // 빈 문자열 == value 지우기

    // input에 초점 맞추기 -> focus()
    input.focus();
}

// input 태그 키가 눌러졌을 때 엔터인 경우를 검사하는 함수
function inputEnter(event){
    
    // console.log(event.key); // 현재 눌러진 키를 반환

    if(event.key == "Enter"){ // 눌러진 key가 Enter인 경우
        readValue(); // 함수 호출
    }
}