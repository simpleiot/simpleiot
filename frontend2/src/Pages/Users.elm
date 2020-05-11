module Pages.Users exposing (Model, Msg, page)

import Dict exposing (Dict)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Generated.Params as Params
import Global
import Http
import Spa.Page
import Spa.Types as Types
import User as U
import Utils.Spa exposing (Page)
import Utils.Styles exposing (palette, size)


page : Page Params.Users Model Msg model msg appMsg
page =
    Spa.Page.component
        { title = always "Users"
        , init = always init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }



-- INIT


type alias Model =
    { users : List U.User
    , userEdits : Dict String U.User
    , error : Maybe Http.Error
    }


init : Params.Users -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ =
    ( { users = []
      , userEdits = Dict.empty
      , error = Nothing
      }
    , Cmd.none
    , Spa.Page.send Global.RequestUsers
    )



-- UPDATE


type Msg
    = PostUser String U.User
    | EditUser String U.User
    | DiscardUserEdits String
    | NewUser


update : Types.PageContext route Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update context msg model =
    case msg of
        EditUser id user ->
            ( { model | userEdits = Dict.insert id user model.userEdits }
            , Cmd.none
            , Cmd.none
            )

        DiscardUserEdits id ->
            ( { model | userEdits = Dict.remove id model.userEdits }
            , Cmd.none
            , Cmd.none
            )

        PostUser _ user ->
            ( model
            , Cmd.none
            , case context.global of
                Global.SignedIn _ ->
                    Spa.Page.send <| Global.UpdateUser user

                Global.SignedOut _ ->
                    Cmd.none
            )

        NewUser ->
            ( { model | users = U.empty :: model.users }
            , Cmd.none
            , Cmd.none
            )



-- SUBSCRIPTIONS


subscriptions : Types.PageContext route Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.none



-- VIEW


view : Types.PageContext route Global.Model -> Model -> Element Msg
view context model =
    case context.global of
        Global.SignedIn sess ->
            column
                [ width fill, spacing 32 ]
                [ el [ padding 16, Font.size 24 ] <| text "Users"
                , el [ padding 16, width fill, Font.bold ] <| button "new user" palette.green NewUser
                , viewUsers model.userEdits sess.data.users
                ]

        _ ->
            el [ padding 16 ] <| text "Sign in to view users."


viewUsers : Dict String U.User -> List U.User -> Element Msg
viewUsers edits users =
    column
        [ width fill
        , spacing 40
        ]
    <|
        List.map
            (\user ->
                viewUser
                    (modified edits user)
                    (userValue edits user)
            )
            users


viewUser : Bool -> U.User -> Element Msg
viewUser modded user =
    wrappedRow
        ([ width fill
         , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
         , Border.color palette.black
         , spacing 6
         ]
            ++ (if modded then
                    [ Background.color palette.orange
                    , below <|
                        buttonRow
                            [ button "discard" palette.pale <| DiscardUserEdits user.id
                            , button "save" palette.green <| PostUser user.id user
                            ]
                    ]

                else
                    []
               )
        )
        [ viewTextProperty
            { name = "First name"
            , value = user.first
            , action = \x -> EditUser user.id { user | first = x }
            }
        , viewTextProperty
            { name = "Last name"
            , value = user.last
            , action = \x -> EditUser user.id { user | last = x }
            }
        , viewTextProperty
            { name = "Email"
            , value = user.email
            , action = \x -> EditUser user.id { user | email = x }
            }
        , viewTextProperty
            { name = "Password"
            , value = user.pass
            , action = \x -> EditUser user.id { user | pass = x }
            }
        ]


buttonRow : List (Element Msg) -> Element Msg
buttonRow =
    row
        [ Font.size 16
        , Font.bold
        , width fill
        , padding 16
        , spacing 16
        ]


button : String -> Color -> Msg -> Element Msg
button lbl color action =
    Input.button
        [ Background.color color
        , padding 16
        , width fill
        , Border.rounded 32
        ]
        { onPress = Just action
        , label = el [ centerX ] <| text lbl
        }


modified : Dict String U.User -> U.User -> Bool
modified edits user =
    case Dict.get user.id edits of
        Just u ->
            u /= user

        Nothing ->
            False


userValue : Dict String U.User -> U.User -> U.User
userValue edits user =
    case Dict.get user.id edits of
        Just u ->
            u

        Nothing ->
            user



-- field : Dict String U.User -> U.User -> String -> String
--field edits user fld =
--    case Dict.get user.id edits of
--        Just u ->
--            fld u
--
--        Nothing ->
--            fld user


type alias TextProperty =
    { name : String
    , value : String
    , action : String -> Msg
    }


viewTextProperty : TextProperty -> Element Msg
viewTextProperty { name, value, action } =
    Input.text
        [ padding 16
        , width (fill |> minimum 150)
        , Border.width 0
        , Border.rounded 0
        , focused [ Background.color palette.yellow ]
        , Background.color palette.pale
        , spacing 0
        ]
        { onChange = action
        , text = value
        , placeholder = Nothing
        , label = label Input.labelAbove name
        }


label : (List (Attribute msg) -> Element msg -> Input.Label msg) -> (String -> Input.Label msg)
label kind =
    kind
        [ padding 16
        , Font.italic
        , Font.color palette.gray
        ]
        << text



--viewRoles =
--    row
--        []
--        << List.map viewRole
--viewRole { role, value, action } =
--    Input.checkbox
--        [ padding 16 ]
--        { checked = value
--        , icon = Input.defaultCheckbox
--        , label = label Input.labelRight role
--        , onChange = action
--        }
